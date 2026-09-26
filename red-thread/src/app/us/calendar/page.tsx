import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { monthsWithPhotos, photosInMonth, toCards } from '@/lib/photos';
import { getSettings } from '@/lib/settings';
import { dayNumber, formatDay, isDay, isMonth, monthGrid, shiftMonth, today } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { PhotoCalendar, type CalendarCell } from '@/components/UsBits';
import type { PhotoCard, Settings } from '@/lib/types';

export const dynamic = 'force-dynamic';

type Props = { searchParams: Promise<{ m?: string }> };

const ROUND_DAYS = new Set([99, 365, 520, 999, 1314, 3344, 5200]);

/** Little labels for days worth remembering. */
function marksFor(day: string, s: Settings) {
  const marks: string[] = [];
  const md = day.slice(5);
  const years = (start: string) => Number(day.slice(0, 4)) - Number(start.slice(0, 4));
  if (isDay(s.togetherSince)) {
    if (day === s.togetherSince) marks.push('在一起');
    else if (md === s.togetherSince.slice(5) && years(s.togetherSince) > 0) marks.push(`${years(s.togetherSince)} 周年`);
    const n = dayNumber(s.togetherSince, day);
    if (n > 0 && (n % 100 === 0 || ROUND_DAYS.has(n))) marks.push(`第 ${n} 天`);
  }
  if (isDay(s.firstMet)) {
    if (day === s.firstMet) marks.push('初遇');
    else if (md === s.firstMet.slice(5) && years(s.firstMet) > 0) marks.push('初遇纪念');
  }
  if (isDay(s.weddingDate) && s.weddingEnabled) {
    if (day === s.weddingDate) marks.push('婚礼');
    else if (md === s.weddingDate.slice(5) && years(s.weddingDate) > 0) marks.push(`结婚 ${years(s.weddingDate)} 周年`);
  }
  if (md === '02-14') marks.push('情人节');
  if (md === '05-20') marks.push('520');
  return marks;
}

export default async function CalendarPage({ searchParams }: Props) {
  await requireSession('/us/calendar');
  const { m } = await searchParams;
  const now = today();
  const month = isMonth(m) ? m : now.slice(0, 7);
  const s = await getSettings();
  const [rows, months] = await Promise.all([photosInMonth(month), monthsWithPhotos()]);
  const cards = await toCards(rows, s, { forUs: true });

  const byDay = new Map<string, PhotoCard[]>();
  rows.forEach((row, i) => {
    const day = row.takenAt!.slice(0, 10);
    byDay.set(day, [...(byDay.get(day) ?? []), cards[i]]);
  });
  const cells: CalendarCell[] = monthGrid(month).map((day) => ({
    day,
    photos: day ? (byDay.get(day) ?? []) : [],
    marks: day ? marksFor(day, s) : [],
    isToday: day === now,
  }));

  const years = [...new Set(months.map((row) => row.month.slice(0, 4)))];

  return (
    <div className="calendar-page">
      <div className="cal-top">
        <Link href={`/us/calendar?m=${shiftMonth(month, -1)}`} className="icon-btn" aria-label="上个月">
          <Icon name="left" />
        </Link>
        <h1 className="cal-title">
          {formatDay(`${month}-01`, 'month')}
          <small>{rows.length > 0 ? `${rows.length} 张照片，${byDay.size} 天有记录` : '这个月还没有照片'}</small>
        </h1>
        <Link href={`/us/calendar?m=${shiftMonth(month, 1)}`} className="icon-btn" aria-label="下个月">
          <Icon name="right" />
        </Link>
      </div>

      <PhotoCalendar cells={cells} />

      {months.length > 0 ? (
        <aside className="cal-index">
          <h2>有照片的月份</h2>
          {years.map((year) => (
            <div key={year} className="cal-year">
              <span className="cal-year-label">{year}</span>
              <div className="cal-months">
                {months
                  .filter((row) => row.month.startsWith(year))
                  .map((row) => (
                    <Link
                      key={row.month}
                      href={`/us/calendar?m=${row.month}`}
                      className={row.month === month ? 'active' : undefined}
                    >
                      {Number(row.month.slice(5))}月<small>{row.n}</small>
                    </Link>
                  ))}
              </div>
            </div>
          ))}
        </aside>
      ) : null}
    </div>
  );
}
