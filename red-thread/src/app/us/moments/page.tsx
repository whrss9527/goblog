import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { MOMENT_KINDS, listMoments, momentPhotoCounts } from '@/lib/moments';
import { firstPhotoPerMoment, getPhotos } from '@/lib/photos';
import { urlFor } from '@/lib/storage';
import { formatRange } from '@/lib/dates';
import { tiltFor } from '@/lib/ui';
import { Icon, KIND_ICON } from '@/components/Icon';
import type { CSSProperties } from 'react';

export const dynamic = 'force-dynamic';

export default async function MomentsPage() {
  await requireSession('/us/moments');
  const [moments, counts, firsts] = await Promise.all([
    listMoments({ newestFirst: true }),
    momentPhotoCounts(),
    firstPhotoPerMoment(),
  ]);
  const covers = new Map(
    (await getPhotos(moments.map((m) => m.coverPhotoId).filter((id): id is string => Boolean(id)))).map((p) => [p.id, p]),
  );
  const label = Object.fromEntries(MOMENT_KINDS.map((k) => [k.value, k.label]));
  const items = await Promise.all(
    moments.map(async (moment) => {
      const cover = (moment.coverPhotoId && covers.get(moment.coverPhotoId)) || firsts.get(moment.id);
      return {
        moment,
        count: counts.get(moment.id) ?? { total: 0, public: 0 },
        thumb: cover ? await urlFor(cover.smKey, cover.visibility === 'private') : null,
        color: cover?.color ?? null,
      };
    }),
  );

  return (
    <div>
      <div className="us-section-head page-head">
        <div>
          <h1>我们的回忆</h1>
          <p className="muted">每一段旅程、每一次约会，都是红线上的一个结。</p>
        </div>
        <Link href="/admin/moments/new" className="btn btn-red">
          <Icon name="plus" size={17} /> 新的回忆
        </Link>
      </div>
      {items.length === 0 ? (
        <div className="empty">
          <p className="hand">还没有写下任何一段回忆</p>
          <Link href="/admin/moments/new" className="btn">
            写下第一段
          </Link>
        </div>
      ) : (
        <div className="moment-grid">
          {items.map(({ moment, count, thumb, color }) => (
            <Link
              key={moment.id}
              href={`/us/moments/${moment.id}`}
              className="moment-card"
              data-reveal
              style={{ '--tilt': tiltFor(moment.id, 1.6) } as CSSProperties}
            >
              <span className="moment-cover" style={{ backgroundColor: color ?? undefined }}>
                {thumb ? <img src={thumb} alt="" loading="lazy" /> : <Icon name={KIND_ICON[moment.kind]} size={36} />}
                {moment.visibility === 'private' ? (
                  <span className="lock-badge" title="只有我们看得到">
                    <Icon name="lock" size={13} />
                  </span>
                ) : null}
              </span>
              <span className="moment-body">
                <span className="chapter-kind">
                  <Icon name={KIND_ICON[moment.kind]} size={13} />
                  {label[moment.kind]}
                </span>
                <strong>{moment.title}</strong>
                <span className="muted small">
                  {formatRange(moment.startsOn, moment.endsOn)}
                  {moment.place ? ` · ${moment.place}` : ''}
                </span>
                <span className="faint small">
                  {count.total} 张照片{count.total > 0 ? `（${count.public} 张公开）` : ''}
                </span>
              </span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
