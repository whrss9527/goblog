import { getSettings } from '@/lib/settings';
import { weddingIcs, weddingReady } from '@/lib/wedding';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  const settings = await getSettings();
  if (!weddingReady(settings)) return new Response('Not found', { status: 404 });
  const url = new URL('/', request.url).toString();
  return new Response(weddingIcs(settings, url), {
    headers: {
      'Content-Type': 'text/calendar; charset=utf-8',
      'Content-Disposition': 'attachment; filename="wedding.ics"',
      'Cache-Control': 'no-store',
    },
  });
}
