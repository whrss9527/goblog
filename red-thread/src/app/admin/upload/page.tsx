import { requireSession } from '@/lib/auth';
import { listMoments } from '@/lib/moments';
import { Uploader } from '@/components/admin/Uploader';

export const dynamic = 'force-dynamic';

export default async function UploadPage({ searchParams }: { searchParams: Promise<{ moment?: string }> }) {
  await requireSession('/admin/upload');
  const [{ moment }, moments] = await Promise.all([searchParams, listMoments({ newestFirst: true })]);
  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>放进新照片</h1>
          <p className="muted small">
            照片会在你的浏览器里压缩成适合浏览的大小，拍摄时间和地点会自动读出来；定位信息只留给我们自己，公开的图片里不会带上。
          </p>
        </div>
      </header>
      <Uploader
        moments={moments.map((m) => ({ id: m.id, title: m.title }))}
        defaultMomentId={moments.some((m) => m.id === moment) ? moment! : null}
      />
    </div>
  );
}
