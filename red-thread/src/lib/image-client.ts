'use client';

/**
 * Everything the uploader does to a photo happens in the browser:
 * read EXIF, decode (HEIC included), make a large and a small JPEG, a tiny
 * blurred preview and an average colour. Re-encoding through a canvas also
 * drops every bit of metadata, so a public photo never leaks where it was
 * taken. The GPS we did read stays in our database, visible only to us.
 */

export type Prepared = {
  lg: Blob;
  sm: Blob;
  width: number;
  height: number;
  blurData: string;
  color: string;
  takenAt: string | null;
  latitude: number | null;
  longitude: number | null;
  camera: string | null;
};

const LG_EDGE = 2400;
const SM_EDGE = 800;

const pad = (n: number) => String(n).padStart(2, '0');
const wallClock = (d: Date) =>
  `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;

const isHeic = (file: File) => /\.(heic|heif)$/i.test(file.name) || /image\/hei[cf]/i.test(file.type);

async function decode(file: File): Promise<ImageBitmap> {
  try {
    return await createImageBitmap(file, { imageOrientation: 'from-image' });
  } catch (error) {
    if (!isHeic(file)) throw error;
    // Chrome and Firefox can't decode HEIC; convert it first (loaded only when needed).
    const { default: heic2any } = await import('heic2any');
    const converted = await heic2any({ blob: file, toType: 'image/jpeg', quality: 0.92 });
    const blob = Array.isArray(converted) ? converted[0] : converted;
    return createImageBitmap(blob, { imageOrientation: 'from-image' });
  }
}

function canvas(width: number, height: number) {
  const el = document.createElement('canvas');
  el.width = width;
  el.height = height;
  const ctx = el.getContext('2d')!;
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = 'high';
  return { el, ctx };
}

function scaled(bitmap: ImageBitmap, edge: number) {
  const ratio = Math.min(1, edge / Math.max(bitmap.width, bitmap.height));
  return { width: Math.round(bitmap.width * ratio), height: Math.round(bitmap.height * ratio) };
}

const toBlob = (el: HTMLCanvasElement, quality: number) =>
  new Promise<Blob>((resolve, reject) =>
    el.toBlob((blob) => (blob ? resolve(blob) : reject(new Error('无法压缩图片'))), 'image/jpeg', quality),
  );

/** Large renditions are drawn in halving steps, which keeps them sharp. */
function drawStepped(bitmap: ImageBitmap, width: number, height: number) {
  let source: CanvasImageSource = bitmap;
  let w = bitmap.width;
  let h = bitmap.height;
  while (w / 2 > width) {
    w = Math.round(w / 2);
    h = Math.round(h / 2);
    const step = canvas(w, h);
    step.ctx.drawImage(source, 0, 0, w, h);
    source = step.el;
  }
  const out = canvas(width, height);
  out.ctx.drawImage(source, 0, 0, width, height);
  return out.el;
}

async function readExif(file: File) {
  try {
    const exifr = (await import('exifr')).default;
    const data = await exifr.parse(file, {
      tiff: true,
      exif: true,
      gps: true,
      pick: ['DateTimeOriginal', 'CreateDate', 'Make', 'Model', 'latitude', 'longitude'],
    });
    if (!data) return null;
    const date: Date | undefined = data.DateTimeOriginal ?? data.CreateDate;
    const camera = [data.Make, data.Model]
      .filter(Boolean)
      .join(' ')
      .replace(/^(\w+) \1 /i, '$1 ')
      .trim();
    return {
      takenAt: date instanceof Date && !Number.isNaN(date.getTime()) ? wallClock(date) : null,
      latitude: typeof data.latitude === 'number' ? data.latitude : null,
      longitude: typeof data.longitude === 'number' ? data.longitude : null,
      camera: camera || null,
    };
  } catch {
    return null;
  }
}

export async function preparePhoto(file: File): Promise<Prepared> {
  const [bitmap, exif] = await Promise.all([decode(file), readExif(file)]);
  try {
    const lgSize = scaled(bitmap, LG_EDGE);
    const smSize = scaled(bitmap, SM_EDGE);
    const lg = await toBlob(drawStepped(bitmap, lgSize.width, lgSize.height), 0.86);
    const sm = await toBlob(drawStepped(bitmap, smSize.width, smSize.height), 0.82);

    const tiny = scaled(bitmap, 24);
    const blur = canvas(tiny.width, tiny.height);
    blur.ctx.drawImage(bitmap, 0, 0, tiny.width, tiny.height);
    const blurData = blur.el.toDataURL('image/jpeg', 0.55);

    const dot = canvas(1, 1);
    dot.ctx.drawImage(blur.el, 0, 0, 1, 1);
    const [r, g, b] = dot.ctx.getImageData(0, 0, 1, 1).data;
    const color = `#${[r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('')}`;

    return {
      lg,
      sm,
      width: lgSize.width,
      height: lgSize.height,
      blurData,
      color,
      takenAt: exif?.takenAt ?? (file.lastModified ? wallClock(new Date(file.lastModified)) : null),
      latitude: exif?.latitude ?? null,
      longitude: exif?.longitude ?? null,
      camera: exif?.camera ?? null,
    };
  } finally {
    bitmap.close();
  }
}

export type Ticket = { via: 'vercel-blob' } | { via: 'put'; url: string; headers: Record<string, string> };

function put(url: string, headers: Record<string, string>, body: Blob, onProgress?: (fraction: number) => void) {
  return new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', url);
    for (const [name, value] of Object.entries(headers)) xhr.setRequestHeader(name, value);
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) onProgress?.(event.loaded / event.total);
    };
    xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`上传失败（${xhr.status}）`)));
    xhr.onerror = () => reject(new Error('上传失败，请检查网络或存储桶的 CORS 设置'));
    xhr.send(body);
  });
}

/** Uploads several files, reporting combined progress. */
export async function uploadFiles(
  files: { key: string; blob: Blob }[],
  onProgress?: (fraction: number) => void,
) {
  const response = await fetch('/api/upload/sign', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ files: files.map((f) => ({ key: f.key, contentType: f.blob.type || 'application/octet-stream' })) }),
  });
  if (!response.ok) throw new Error((await response.json().catch(() => null))?.error ?? '无法获取上传地址');
  const { tickets } = (await response.json()) as { tickets: Ticket[] };

  const total = files.reduce((sum, f) => sum + f.blob.size, 0) || 1;
  const done = files.map(() => 0);
  const report = () => onProgress?.(done.reduce((a, b) => a + b, 0) / total);

  await Promise.all(
    files.map(async (file, i) => {
      const ticket = tickets[i];
      const track = (fraction: number) => {
        done[i] = fraction * file.blob.size;
        report();
      };
      if (ticket.via === 'vercel-blob') {
        const { upload } = await import('@vercel/blob/client');
        await upload(file.key, file.blob, {
          access: 'public',
          handleUploadUrl: '/api/upload/blob',
          contentType: file.blob.type,
          onUploadProgress: ({ loaded, total: size }) => track(size ? loaded / size : 0),
        });
      } else {
        await put(ticket.url, ticket.headers, file.blob, track);
      }
      track(1);
    }),
  );
}
