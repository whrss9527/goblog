import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { listMoments } from '@/lib/moments';
import { listPhotos, toCards, type PhotoFilter } from '@/lib/photos';
import { getSettings } from '@/lib/settings';
import { hasDemo } from '@/lib/demo';
import { STORAGE_LABEL, STORAGE } from '@/lib/storage';
import { clearDemoAction, loadDemoAction } from '@/app/admin/actions';
import { PhotoManager, type AdminPhoto } from '@/components/admin/PhotoManager';
import { ConfirmSubmit } from '@/components/admin/Small';
import { Icon } from '@/components/Icon';

export const dynamic = 'force-dynamic';

const PAGE = 120;
const FILTERS: { key: string; label: string; filter: PhotoFilter }[] = [
  { key: 'recent', label: '最近上传', filter: { order: 'uploaded' } },
  { key: 'all', label: '按拍摄时间', filter: { order: 'newest' } },
  { key: 'private', label: '只给彼此', filter: { visibility: 'private', order: 'newest' } },
  { key: 'public', label: '公开的', filter: { visibility: 'public', order: 'newest' } },
  { key: 'featured', label: '精选', filter: { featured: true, order: 'newest' } },
  { key: 'loose', label: '未归档', filter: { unassigned: true, order: 'newest' } },
];

type Props = { searchParams: Promise<{ filter?: string; moment?: string; page?: string }> };

export default async function AdminPhotos({ searchParams }: Props) {
  await requireSession('/admin/photos');
  const params = await searchParams;
  const active = FILTERS.find((f) => f.key === params.filter) ?? FILTERS[0];
  const page = Math.max(1, Number(params.page) || 1);
  const [settings, moments, demo] = await Promise.all([getSettings(), listMoments({ newestFirst: true }), hasDemo()]);
  const momentFilter = moments.find((m) => m.id === params.moment);
  const rows = await listPhotos({
    ...active.filter,
    ...(momentFilter ? { momentId: momentFilter.id } : {}),
    limit: PAGE + 1,
    offset: (page - 1) * PAGE,
  });
  const cards = await toCards(rows.slice(0, PAGE), settings, { forUs: true });
  const titles = new Map(moments.map((m) => [m.id, m.title]));
  const photos: AdminPhoto[] = cards.map((card, i) => ({
    ...card,
    visibility: rows[i].visibility,
    featured: rows[i].featured,
    favorite: rows[i].favorite,
    momentTitle: rows[i].momentId ? (titles.get(rows[i].momentId!) ?? null) : null,
  }));
  const href = (key: string, p = 1) =>
    `/admin/photos?filter=${key}${momentFilter ? `&moment=${momentFilter.id}` : ''}${p > 1 ? `&page=${p}` : ''}`;

  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>照片</h1>
          <p className="muted small">存储：{STORAGE_LABEL[STORAGE]}</p>
        </div>
        <Link href="/admin/upload" className="btn btn-red">
          <Icon name="upload" size={17} /> 上传照片
        </Link>
      </header>

      <div className="admin-filters">
        <nav className="chips">
          {FILTERS.map((f) => (
            <Link key={f.key} href={href(f.key)} className={f.key === active.key ? 'chip active' : 'chip'}>
              {f.label}
            </Link>
          ))}
        </nav>
        {momentFilter ? (
          <span className="chip active">
            回忆：{momentFilter.title} <Link href={`/admin/photos?filter=${active.key}`}>×</Link>
          </span>
        ) : null}
      </div>

      {photos.length > 0 ? (
        <PhotoManager photos={photos} moments={moments.map((m) => ({ id: m.id, title: m.title }))} />
      ) : (
        <div className="empty card">
          <p className="hand">这里还空空的</p>
          <p>上传第一张照片吧。也可以先用示例数据看看整本相册的样子。</p>
          <div className="row">
            <Link href="/admin/upload" className="btn btn-red">
              <Icon name="upload" size={17} /> 上传照片
            </Link>
            {!demo ? (
              <form action={loadDemoAction}>
                <button type="submit" className="btn">
                  <Icon name="sparkles" size={17} /> 用示例数据看看效果
                </button>
              </form>
            ) : null}
          </div>
        </div>
      )}

      {rows.length > PAGE || page > 1 ? (
        <div className="pager">
          {page > 1 ? (
            <Link href={href(active.key, page - 1)} className="btn btn-sm">
              ← 上一页
            </Link>
          ) : (
            <span />
          )}
          {rows.length > PAGE ? (
            <Link href={href(active.key, page + 1)} className="btn btn-sm">
              下一页 →
            </Link>
          ) : null}
        </div>
      ) : null}

      {demo ? (
        <form action={clearDemoAction} className="demo-bar card">
          <span>
            <Icon name="sparkles" size={16} /> 现在相册里有示例数据（插画照片和几段示例回忆）。
          </span>
          <ConfirmSubmit message="清除所有示例照片和示例回忆？你自己上传的照片不受影响。" className="btn btn-sm">
            清除示例数据
          </ConfirmSubmit>
        </form>
      ) : null}
    </div>
  );
}
