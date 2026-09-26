import Link from 'next/link';
import { notFound } from 'next/navigation';
import { requireSession } from '@/lib/auth';
import { getMoment } from '@/lib/moments';
import { listPhotos } from '@/lib/photos';
import { urlFor } from '@/lib/storage';
import { deleteMomentAction } from '@/app/admin/actions';
import { ConfirmSubmit } from '@/components/admin/Small';
import { Icon } from '@/components/Icon';
import { MomentForm } from '../MomentForm';

export const dynamic = 'force-dynamic';

export default async function EditMoment({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  await requireSession(`/admin/moments/${id}`);
  const moment = await getMoment(id);
  if (!moment) notFound();
  const photos = await listPhotos({ momentId: id });
  const covers = await Promise.all(
    photos.slice(0, 60).map(async (p) => ({ id: p.id, thumb: await urlFor(p.smKey, p.visibility === 'private') })),
  );
  return (
    <div>
      <Link href="/admin/moments" className="muted back-link">
        ← 所有章节
      </Link>
      <header className="admin-head">
        <div>
          <h1>{moment.title}</h1>
          <p className="muted small">{photos.length} 张照片</p>
        </div>
        <div className="row">
          <Link href={`/admin/photos?filter=all&moment=${moment.id}`} className="btn btn-sm">
            <Icon name="image" size={15} /> 整理这些照片
          </Link>
          <Link href={`/admin/upload?moment=${moment.id}`} className="btn btn-sm">
            <Icon name="upload" size={15} /> 往里放照片
          </Link>
        </div>
      </header>
      <MomentForm moment={moment} covers={covers} />
      <form action={deleteMomentAction.bind(null, moment.id)} className="danger-zone">
        <ConfirmSubmit message="删除这段回忆？里面的照片不会被删除，只是变成“未归档”。">删除这段回忆</ConfirmSubmit>
      </form>
    </div>
  );
}
