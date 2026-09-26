import { handleUpload, type HandleUploadBody } from '@vercel/blob/client';
import { getSession } from '@/lib/auth';
import { isStorageKey } from '@/lib/ids';

/** Token endpoint for Vercel Blob client uploads (same flow as exif-photo-blog). */
export async function POST(request: Request) {
  const body = (await request.json()) as HandleUploadBody;
  try {
    const result = await handleUpload({
      body,
      request,
      onBeforeGenerateToken: async (pathname) => {
        if (!(await getSession())) throw new Error('需要先登录');
        if (!isStorageKey(pathname)) throw new Error('Invalid key');
        return {
          allowedContentTypes: ['image/*', 'audio/*'],
          addRandomSuffix: false,
          maximumSizeInBytes: 60 * 1024 * 1024,
          cacheControlMaxAge: 60 * 60 * 24 * 365,
        };
      },
    });
    return Response.json(result);
  } catch (error) {
    return Response.json({ error: (error as Error).message }, { status: 400 });
  }
}
