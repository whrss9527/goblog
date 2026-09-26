import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { countPhotos, listPhotos, photosOnDays, toCards } from '@/lib/photos';
import { getSettings, partnerName } from '@/lib/settings';
import {
  TIMEZONE,
  daysBetween,
  dayNumber,
  formatDay,
  isDay,
  monthDaysAround,
  today,
  upcomingMilestones,
  yearsAgoLabel,
} from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { PhotoMasonry, PolaroidStrip } from '@/components/PhotoGroups';
import { DrawMemoryButton } from '@/components/UsBits';
import type { Photo } from '@/lib/types';

export const dynamic = 'force-dynamic';

function greeting() {
  const hour = Number(
    new Intl.DateTimeFormat('en-US', { timeZone: TIMEZONE, hour: 'numeric', hourCycle: 'h23' }).format(new Date()),
  );
  if (hour < 5) return '夜深了';
  if (hour < 11) return '早安';
  if (hour < 14) return '午安';
  if (hour < 18) return '下午好';
  return '晚上好';
}

const SWEET = [
  '今天也要记得牵手。',
  '{other}说，今天的你也很好看。',
  '有空的话，一起去拍张照片吧。',
  '今天的晚饭，要不要一起做？',
  '记得抱一抱{other}。',
  '把今天也过成值得放进相册的一天吧。',
  '偷偷告诉你：{other}又在翻你们的旧照片了。',
];

function groupByYear(photos: Photo[]) {
  const groups = new Map<number, Photo[]>();
  for (const photo of photos) {
    const year = Number(photo.takenAt!.slice(0, 4));
    groups.set(year, [...(groups.get(year) ?? []), photo]);
  }
  return [...groups.entries()].sort((a, b) => b[0] - a[0]);
}

export default async function UsHome() {
  const session = await requireSession('/us');
  const s = await getSettings();
  const now = today();
  const year = Number(now.slice(0, 4));

  const [exact, recentRows, favoriteRows, counts] = await Promise.all([
    photosOnDays([now.slice(5)], year),
    listPhotos({ order: 'uploaded', limit: 12 }),
    listPhotos({ favorite: true, order: 'newest', limit: 12 }),
    countPhotos(),
  ]);
  const nearby = exact.length === 0;
  const memories = nearby ? await photosOnDays(monthDaysAround(now, 3), year) : exact;
  const years = groupByYear(memories);
  const cards = new Map(
    (await toCards([...memories, ...recentRows, ...favoriteRows], s, { forUs: true })).map((c) => [c.id, c]),
  );
  const pick = (rows: Photo[]) => rows.map((row) => cards.get(row.id)!);

  const me = partnerName(s, session.partner) ?? '';
  const other = partnerName(s, session.partner === 'a' ? 'b' : 'a') ?? 'Ta';
  const since = isDay(s.togetherSince) ? s.togetherSince : null;
  const day = since ? dayNumber(since, now) : null;
  const milestones = since ? upcomingMilestones(since, now) : [];
  if (s.weddingEnabled && isDay(s.weddingDate) && s.weddingDate >= now) {
    milestones.unshift({ label: '我们的婚礼', day: s.weddingDate, inDays: daysBetween(now, s.weddingDate) });
  }
  const sweet = SWEET[(day ?? 0) % SWEET.length].replaceAll('{other}', other);

  return (
    <div className="us-home">
      <section className="welcome" data-reveal>
        <div className="welcome-text">
          <p className="welcome-hi hand">
            {greeting()}，{me}
          </p>
          {day ? (
            <h1 className="welcome-day">
              今天是我们在一起的第 <b>{day.toLocaleString('zh-CN')}</b> 天
            </h1>
          ) : (
            <h1 className="welcome-day">
              <Link href="/admin/settings">先去设置我们在一起的日子吧 →</Link>
            </h1>
          )}
          <p className="muted">{sweet}</p>
          <div className="welcome-actions">
            <DrawMemoryButton />
            <Link href="/admin/upload" className="btn">
              <Icon name="upload" size={17} /> 放进新照片
            </Link>
          </div>
        </div>
        {milestones.length > 0 ? (
          <ul className="milestones">
            {milestones.slice(0, 3).map((m) => (
              <li key={m.label + m.day}>
                <span className="milestone-in">
                  {m.inDays === 0 ? '今天' : (
                    <>
                      <b>{m.inDays}</b> 天后
                    </>
                  )}
                </span>
                <span className="milestone-label">{m.label}</span>
                <span className="milestone-day">{formatDay(m.day)}</span>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="us-section" data-reveal>
        <div className="us-section-head">
          <h2>{nearby ? '那些年的这几天' : '那年今日'}</h2>
          <p className="muted">{formatDay(now)}</p>
        </div>
        {years.length > 0 ? (
          years.map(([y, photos]) => (
            <div key={y} className="otd">
              <p className="otd-label hand">
                {nearby ? `${year - y} 年前的这几天` : yearsAgoLabel(y, year)}
                <span className="faint"> · {y}</span>
              </p>
              <PolaroidStrip photos={pick(photos)} />
            </div>
          ))
        ) : (
          <div className="empty">
            <p className="hand">往年的今天还没有照片</p>
            <p>那就从今天开始，拍一张明年会想翻出来看的照片吧。</p>
          </div>
        )}
      </section>

      {favoriteRows.length > 0 ? (
        <section className="us-section" data-reveal>
          <div className="us-section-head">
            <h2>我们最爱的</h2>
            <Link href="/us/photos?filter=favorite" className="muted">
              全部 →
            </Link>
          </div>
          <PolaroidStrip photos={pick(favoriteRows)} />
        </section>
      ) : null}

      <section className="us-section" data-reveal>
        <div className="us-section-head">
          <h2>最近放进来的</h2>
          <p className="muted">
            一共 {counts.total} 张 · <Icon name="lock" size={13} /> {counts.private} 张只给彼此 · {counts.public} 张公开
          </p>
        </div>
        {recentRows.length > 0 ? (
          <PhotoMasonry photos={pick(recentRows)} />
        ) : (
          <div className="empty">
            <p className="hand">相册还是空的</p>
            <Link href="/admin/upload" className="btn btn-red">
              <Icon name="upload" size={17} /> 放进第一张照片
            </Link>
          </div>
        )}
      </section>
    </div>
  );
}
