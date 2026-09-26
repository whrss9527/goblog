import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { MOMENT_KINDS, listMoments, momentPhotoCounts } from '@/lib/moments';
import { formatRange } from '@/lib/dates';
import { Icon, KIND_ICON } from '@/components/Icon';

export const dynamic = 'force-dynamic';

export default async function AdminMoments() {
  await requireSession('/admin/moments');
  const [moments, counts] = await Promise.all([listMoments({ newestFirst: true }), momentPhotoCounts()]);
  const label = Object.fromEntries(MOMENT_KINDS.map((k) => [k.value, k.label]));
  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>回忆章节</h1>
          <p className="muted small">每段回忆都是红线上的一个结。公开的回忆会按时间顺序出现在对外的故事里。</p>
        </div>
        <Link href="/admin/moments/new" className="btn btn-red">
          <Icon name="plus" size={17} /> 新的回忆
        </Link>
      </header>
      {moments.length === 0 ? (
        <div className="empty card">
          <p className="hand">还没有章节</p>
          <p>比如“第一次见面”“第一次一起旅行”“求婚”，都可以是一个章节。</p>
        </div>
      ) : (
        <ul className="rows card">
          {moments.map((moment) => {
            const count = counts.get(moment.id) ?? { total: 0, public: 0 };
            return (
              <li key={moment.id}>
                <Link href={`/admin/moments/${moment.id}`} className="row-link">
                  <span className="row-icon">
                    <Icon name={KIND_ICON[moment.kind]} size={18} />
                  </span>
                  <span className="row-main">
                    <strong>{moment.title}</strong>
                    <span className="faint small">
                      {label[moment.kind]} · {formatRange(moment.startsOn, moment.endsOn)}
                      {moment.place ? ` · ${moment.place}` : ''}
                    </span>
                  </span>
                  <span className="row-side small">
                    {count.total} 张 · {count.public} 公开
                  </span>
                  <span className={`row-tag ${moment.visibility}`}>
                    <Icon name={moment.visibility === 'public' ? 'eye' : 'lock'} size={13} />
                    {moment.visibility === 'public' ? '公开' : '私密'}
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
