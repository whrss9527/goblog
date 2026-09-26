import { requireSession } from '@/lib/auth';
import { getPhotos } from '@/lib/photos';
import { urlFor } from '@/lib/storage';
import { MomentForm } from '../MomentForm';

export const dynamic = 'force-dynamic';

/** `?photos=a,b,c` pre-fills the dates and place from the selected photos. */
export default async function NewMoment({ searchParams }: { searchParams: Promise<{ photos?: string }> }) {
  await requireSession('/admin/moments/new');
  const ids = ((await searchParams).photos ?? '')
    .split(',')
    .filter((id) => /^[a-z0-9-]{1,40}$/.test(id))
    .slice(0, 300);
  const photos = await getPhotos(ids);
  const days = photos.map((p) => p.takenAt?.slice(0, 10)).filter((d): d is string => Boolean(d)).sort();
  const places = photos.map((p) => p.place).filter((p): p is string => Boolean(p));
  const place = places.sort((a, b) => places.filter((p) => p === b).length - places.filter((p) => p === a).length)[0];
  const covers = await Promise.all(
    photos.slice(0, 24).map(async (p) => ({ id: p.id, thumb: await urlFor(p.smKey, p.visibility === 'private') })),
  );

  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>新的回忆</h1>
          {photos.length ? <p className="muted small">会把选中的 {photos.length} 张照片放进这段回忆。</p> : null}
        </div>
      </header>
      <MomentForm
        moment={null}
        defaults={{
          startsOn: days[0] ?? new Date().toISOString().slice(0, 10),
          endsOn: days.length > 1 && days[days.length - 1] !== days[0] ? days[days.length - 1] : null,
          place: place ?? null,
          visibility: 'private',
        }}
        covers={covers}
        photoIds={photos.map((p) => p.id)}
      />
    </div>
  );
}
