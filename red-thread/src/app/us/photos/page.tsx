import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { listPhotos, toCards, type PhotoFilter } from '@/lib/photos';
import { getSettings } from '@/lib/settings';
import { PhotoMasonry } from '@/components/PhotoGroups';

export const dynamic = 'force-dynamic';

const PAGE = 60;
const FILTERS: { key: string; label: string; filter: PhotoFilter }[] = [
  { key: 'all', label: '全部', filter: {} },
  { key: 'private', label: '只给彼此', filter: { visibility: 'private' } },
  { key: 'public', label: '公开的', filter: { visibility: 'public' } },
  { key: 'favorite', label: '最爱', filter: { favorite: true } },
  { key: 'loose', label: '还没归档', filter: { unassigned: true } },
];

type Props = { searchParams: Promise<{ filter?: string; page?: string }> };

export default async function PhotosPage({ searchParams }: Props) {
  await requireSession('/us/photos');
  const params = await searchParams;
  const active = FILTERS.find((f) => f.key === params.filter) ?? FILTERS[0];
  const page = Math.max(1, Number(params.page) || 1);
  const [settings, rows] = await Promise.all([
    getSettings(),
    listPhotos({ ...active.filter, order: 'newest', limit: PAGE + 1, offset: (page - 1) * PAGE }),
  ]);
  const hasMore = rows.length > PAGE;
  const cards = await toCards(rows.slice(0, PAGE), settings, { forUs: true });
  const href = (key: string, p = 1) => `/us/photos?filter=${key}${p > 1 ? `&page=${p}` : ''}`;

  return (
    <div>
      <div className="us-section-head page-head">
        <h1>所有照片</h1>
        <nav className="chips">
          {FILTERS.map((f) => (
            <Link key={f.key} href={href(f.key)} className={f.key === active.key ? 'chip active' : 'chip'}>
              {f.label}
            </Link>
          ))}
        </nav>
      </div>
      {cards.length > 0 ? (
        <PhotoMasonry photos={cards} />
      ) : (
        <div className="empty">
          <p className="hand">这里还空空的</p>
        </div>
      )}
      {page > 1 || hasMore ? (
        <div className="pager">
          {page > 1 ? (
            <Link href={href(active.key, page - 1)} className="btn btn-sm">
              ← 新一些的
            </Link>
          ) : (
            <span />
          )}
          {hasMore ? (
            <Link href={href(active.key, page + 1)} className="btn btn-sm">
              更早的 →
            </Link>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
