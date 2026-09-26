import 'server-only';
import path from 'node:path';
import fs from 'node:fs/promises';
import type { S3Client, S3ClientConfig } from '@aws-sdk/client-s3';
import { DEMO_KEY } from './ids';

/**
 * Same storage providers and environment variables as exif-photo-blog
 * (Vercel Blob, Cloudflare R2, AWS S3, MinIO), plus any S3-compatible bucket
 * (阿里云 OSS, 腾讯云 COS …) and a local folder for development / Docker.
 *
 * Files are uploaded straight from the browser, so the server never has to
 * carry photo bytes; the database only stores the storage *key*.
 */
export type StorageKind =
  | 'vercel-blob'
  | 'cloudflare-r2'
  | 'aws-s3'
  | 'minio'
  | 's3-compatible'
  | 'local';

const env = process.env;
const stripProtocol = (value?: string) => value?.replace(/^https?:\/\//, '').replace(/\/+$/, '');

const BLOB_STORE_ID = env.BLOB_READ_WRITE_TOKEN?.match(
  /^vercel_blob_rw_([a-z0-9]+)_[a-z0-9]+$/i,
)?.[1]?.toLowerCase();

const configured: Record<StorageKind, boolean> = {
  'vercel-blob': Boolean(BLOB_STORE_ID),
  'cloudflare-r2': Boolean(
    env.NEXT_PUBLIC_CLOUDFLARE_R2_BUCKET &&
      env.NEXT_PUBLIC_CLOUDFLARE_R2_ACCOUNT_ID &&
      env.CLOUDFLARE_R2_ACCESS_KEY &&
      env.CLOUDFLARE_R2_SECRET_ACCESS_KEY,
  ),
  'aws-s3': Boolean(
    env.NEXT_PUBLIC_AWS_S3_BUCKET &&
      env.NEXT_PUBLIC_AWS_S3_REGION &&
      env.AWS_S3_ACCESS_KEY &&
      env.AWS_S3_SECRET_ACCESS_KEY,
  ),
  minio: Boolean(
    env.NEXT_PUBLIC_MINIO_BUCKET &&
      env.NEXT_PUBLIC_MINIO_DOMAIN &&
      env.MINIO_ACCESS_KEY &&
      env.MINIO_SECRET_ACCESS_KEY,
  ),
  's3-compatible': Boolean(
    env.S3_ENDPOINT && env.S3_BUCKET && env.S3_ACCESS_KEY && env.S3_SECRET_ACCESS_KEY,
  ),
  local: true,
};

function pickStorage(): StorageKind {
  const preferred = env.NEXT_PUBLIC_STORAGE_PREFERENCE as StorageKind | undefined;
  if (preferred && configured[preferred]) return preferred;
  const order: StorageKind[] = ['vercel-blob', 'cloudflare-r2', 'aws-s3', 'minio', 's3-compatible'];
  return order.find((kind) => configured[kind]) ?? 'local';
}

export const STORAGE: StorageKind = pickStorage();

export const STORAGE_LABEL: Record<StorageKind, string> = {
  'vercel-blob': 'Vercel Blob',
  'cloudflare-r2': 'Cloudflare R2',
  'aws-s3': 'AWS S3',
  minio: 'MinIO',
  's3-compatible': 'S3 兼容存储',
  local: '本地磁盘',
};

/** `STORAGE_SIGNED_URLS=1`: the bucket is private, every read gets a signed URL. */
const SIGN_EVERYTHING = env.STORAGE_SIGNED_URLS === '1';

type S3Target = {
  bucket: string;
  client: () => Promise<S3Client>;
  /** Base URL for public reads; undefined means "always sign". */
  publicBase?: string;
};

let s3ClientCache: Promise<S3Client> | undefined;
function s3Client(options: S3ClientConfig) {
  return () => {
    s3ClientCache ??= import('@aws-sdk/client-s3').then(
      ({ S3Client: Client }) => new Client(options),
    );
    return s3ClientCache;
  };
}

function s3Target(): S3Target | null {
  switch (STORAGE) {
    case 'cloudflare-r2': {
      const domain = stripProtocol(env.NEXT_PUBLIC_CLOUDFLARE_R2_PUBLIC_DOMAIN);
      return {
        bucket: env.NEXT_PUBLIC_CLOUDFLARE_R2_BUCKET!,
        publicBase: domain ? `https://${domain}` : undefined,
        client: s3Client({
          region: 'auto',
          endpoint: `https://${env.NEXT_PUBLIC_CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com`,
          credentials: {
            accessKeyId: env.CLOUDFLARE_R2_ACCESS_KEY!,
            secretAccessKey: env.CLOUDFLARE_R2_SECRET_ACCESS_KEY!,
          },
        }),
      };
    }
    case 'aws-s3': {
      const bucket = env.NEXT_PUBLIC_AWS_S3_BUCKET!;
      const region = env.NEXT_PUBLIC_AWS_S3_REGION!;
      return {
        bucket,
        publicBase: `https://${bucket}.s3.${region}.amazonaws.com`,
        client: s3Client({
          region,
          credentials: {
            accessKeyId: env.AWS_S3_ACCESS_KEY!,
            secretAccessKey: env.AWS_S3_SECRET_ACCESS_KEY!,
          },
        }),
      };
    }
    case 'minio': {
      const protocol = env.NEXT_PUBLIC_MINIO_DISABLE_SSL === '1' ? 'http' : 'https';
      const port = env.NEXT_PUBLIC_MINIO_PORT ? `:${env.NEXT_PUBLIC_MINIO_PORT}` : '';
      const endpoint = `${protocol}://${stripProtocol(env.NEXT_PUBLIC_MINIO_DOMAIN)}${port}`;
      const bucket = env.NEXT_PUBLIC_MINIO_BUCKET!;
      return {
        bucket,
        publicBase: `${endpoint}/${bucket}`,
        client: s3Client({
          region: 'us-east-1',
          endpoint,
          forcePathStyle: true,
          credentials: {
            accessKeyId: env.MINIO_ACCESS_KEY!,
            secretAccessKey: env.MINIO_SECRET_ACCESS_KEY!,
          },
        }),
      };
    }
    case 's3-compatible': {
      const base = env.S3_PUBLIC_BASE_URL?.replace(/\/+$/, '');
      return {
        bucket: env.S3_BUCKET!,
        publicBase: base || undefined,
        client: s3Client({
          region: env.S3_REGION || 'auto',
          endpoint: env.S3_ENDPOINT,
          forcePathStyle: env.S3_FORCE_PATH_STYLE === '1',
          credentials: {
            accessKeyId: env.S3_ACCESS_KEY!,
            secretAccessKey: env.S3_SECRET_ACCESS_KEY!,
          },
        }),
      };
    }
    default:
      return null;
  }
}

const S3 = s3Target();

async function presign(key: string, method: 'GET' | 'PUT', contentType?: string) {
  const [{ GetObjectCommand, PutObjectCommand }, { getSignedUrl }] = await Promise.all([
    import('@aws-sdk/client-s3'),
    import('@aws-sdk/s3-request-presigner'),
  ]);
  const client = await S3!.client();
  if (method === 'PUT') {
    const command = new PutObjectCommand({
      Bucket: S3!.bucket,
      Key: key,
      ContentType: contentType,
      CacheControl: 'public, max-age=31536000, immutable',
    });
    return getSignedUrl(client, command, { expiresIn: 60 * 15 });
  }
  // Sign from the start of the UTC day, valid for two days: every visitor gets
  // the same URL all day long, so browsers and CDNs can actually cache it.
  const day = new Date();
  day.setUTCHours(0, 0, 0, 0);
  const command = new GetObjectCommand({ Bucket: S3!.bucket, Key: key });
  return getSignedUrl(client, command, { expiresIn: 60 * 60 * 48, signingDate: day });
}

/**
 * Resolve a stored key to something an <img> can load.
 * `isPrivate` photos never use a public bucket domain when signing is possible.
 */
export async function urlFor(key: string, isPrivate: boolean): Promise<string> {
  if (/^https?:\/\//.test(key)) return key;
  if (DEMO_KEY.test(key)) return `/${key}`;
  switch (STORAGE) {
    case 'vercel-blob':
      return `https://${BLOB_STORE_ID}.public.blob.vercel-storage.com/${key}`;
    case 'local':
      return `/files/${key}`;
    default:
      if (!isPrivate && !SIGN_EVERYTHING && S3?.publicBase) return `${S3.publicBase}/${key}`;
      return presign(key, 'GET');
  }
}

export type UploadTicket =
  | { via: 'vercel-blob' }
  | { via: 'put'; url: string; headers: Record<string, string> };

/** How the browser should upload a file to `key`. */
export async function uploadTicket(key: string, contentType: string): Promise<UploadTicket> {
  switch (STORAGE) {
    case 'vercel-blob':
      return { via: 'vercel-blob' };
    case 'local':
      return {
        via: 'put',
        url: `/api/upload/local?key=${encodeURIComponent(key)}`,
        headers: { 'Content-Type': contentType },
      };
    default:
      return {
        via: 'put',
        url: await presign(key, 'PUT', contentType),
        headers: {
          'Content-Type': contentType,
          'Cache-Control': 'public, max-age=31536000, immutable',
        },
      };
  }
}

export function localPath(key: string) {
  const root = path.resolve(
    /*turbopackIgnore: true*/ env.LOCAL_STORAGE_DIR || path.join(process.cwd(), '.data', 'uploads'),
  );
  const file = path.resolve(/*turbopackIgnore: true*/ root, key);
  if (!file.startsWith(root + path.sep)) throw new Error('Invalid key');
  return file;
}

export async function deleteKeys(keys: (string | null | undefined)[]) {
  const real = keys.filter((key): key is string => Boolean(key) && !DEMO_KEY.test(key!));
  if (real.length === 0) return;
  switch (STORAGE) {
    case 'vercel-blob': {
      const { del } = await import('@vercel/blob');
      await del(real.map((key) => `https://${BLOB_STORE_ID}.public.blob.vercel-storage.com/${key}`));
      return;
    }
    case 'local':
      await Promise.all(real.map((key) => fs.rm(localPath(key), { force: true })));
      return;
    default: {
      const { DeleteObjectsCommand } = await import('@aws-sdk/client-s3');
      const client = await S3!.client();
      await client.send(
        new DeleteObjectsCommand({
          Bucket: S3!.bucket,
          Delete: { Objects: real.map((Key) => ({ Key })) },
        }),
      );
    }
  }
}

/** Shown on the admin dashboard so a misconfiguration is obvious. */
export function storageWarning(): string | null {
  if (STORAGE === 'local' && env.VERCEL) {
    return '当前部署在 Vercel 上却在使用本地磁盘存储，照片会在下次部署时丢失。请配置 Vercel Blob / R2 / S3 等存储。';
  }
  return null;
}
