import fs from 'node:fs/promises';
import path from 'node:path';
import { getSession } from '@/lib/auth';
import { getSettings } from '@/lib/settings';
import { findPhotoByKey } from '@/lib/photos';
import { STORAGE, localPath } from '@/lib/storage';
import { isStorageKey } from '@/lib/ids';

const TYPES: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.gif': 'image/gif',
  '.avif': 'image/avif',
  '.heic': 'image/heic',
  '.heif': 'image/heif',
  '.mp3': 'audio/mpeg',
  '.m4a': 'audio/mp4',
  '.aac': 'audio/aac',
  '.ogg': 'audio/ogg',
  '.wav': 'audio/wav',
};

/**
 * Serves files for the local-disk storage (development and Docker).
 * Private photos, and every original, are only handed to the two of us.
 */
export async function GET(_request: Request, { params }: { params: Promise<{ key: string[] }> }) {
  if (STORAGE !== 'local') return new Response('Not found', { status: 404 });
  const key = (await params).key.join('/');
  if (!isStorageKey(key)) return new Response('Not found', { status: 404 });

  let cacheControl = 'public, max-age=31536000, immutable';
  if (key.startsWith('photos/')) {
    const photo = await findPhotoByKey(key);
    if (!photo) return new Response('Not found', { status: 404 });
    if (photo.visibility === 'private' || key === photo.origKey) {
      if (!(await getSession())) return new Response('Not found', { status: 404 });
      cacheControl = 'private, max-age=86400';
    }
  } else {
    // Media (music) is only public once it is the album's soundtrack.
    const settings = await getSettings();
    if (settings.music !== key && !(await getSession())) return new Response('Not found', { status: 404 });
  }

  try {
    const body = await fs.readFile(localPath(key));
    return new Response(body, {
      headers: {
        'Content-Type': TYPES[path.extname(key).toLowerCase()] ?? 'application/octet-stream',
        'Cache-Control': cacheControl,
      },
    });
  } catch {
    return new Response('Not found', { status: 404 });
  }
}
