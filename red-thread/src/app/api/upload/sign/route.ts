import { getSession } from '@/lib/auth';
import { isStorageKey } from '@/lib/ids';
import { uploadTicket } from '@/lib/storage';

type Body = { files?: { key?: string; contentType?: string }[] };

/** Hands the browser one upload ticket per file (presigned PUT, local PUT or "use Vercel Blob"). */
export async function POST(request: Request) {
  if (!(await getSession())) return Response.json({ error: '需要先登录' }, { status: 401 });
  const body = (await request.json().catch(() => ({}))) as Body;
  const files = body.files ?? [];
  if (files.length === 0 || files.length > 12) return Response.json({ error: 'Bad request' }, { status: 400 });
  for (const file of files) {
    if (!file.key || !isStorageKey(file.key)) return Response.json({ error: 'Invalid key' }, { status: 400 });
    if (!file.contentType || !/^(image|audio)\/[\w.+-]+$/.test(file.contentType)) {
      return Response.json({ error: 'Invalid content type' }, { status: 400 });
    }
  }
  const tickets = await Promise.all(files.map((file) => uploadTicket(file.key!, file.contentType!)));
  return Response.json({ tickets });
}
