import fs from 'node:fs/promises';
import path from 'node:path';
import { getSession } from '@/lib/auth';
import { isStorageKey } from '@/lib/ids';
import { STORAGE, localPath } from '@/lib/storage';

const MAX_BYTES = 60 * 1024 * 1024;

/** Upload target for the local-disk storage: `PUT /api/upload/local?key=…`. */
export async function PUT(request: Request) {
  if (STORAGE !== 'local') return new Response('Local storage is not in use', { status: 404 });
  if (!(await getSession())) return new Response('Unauthorized', { status: 401 });

  const key = new URL(request.url).searchParams.get('key') ?? '';
  if (!isStorageKey(key)) return new Response('Invalid key', { status: 400 });
  const length = Number(request.headers.get('content-length') ?? 0);
  if (length > MAX_BYTES) return new Response('Too large', { status: 413 });

  const body = Buffer.from(await request.arrayBuffer());
  if (body.byteLength > MAX_BYTES) return new Response('Too large', { status: 413 });
  const file = localPath(key);
  await fs.mkdir(path.dirname(file), { recursive: true });
  await fs.writeFile(file, body);
  return new Response(null, { status: 204 });
}
