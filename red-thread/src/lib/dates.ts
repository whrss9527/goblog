/**
 * Calendar maths on plain `YYYY-MM-DD` strings, so a date never shifts
 * because the server happens to run in UTC. Safe for client and server.
 */

export const TIMEZONE = process.env.NEXT_PUBLIC_TIMEZONE || 'Asia/Shanghai';

const DAY_MS = 86_400_000;
const WEEKDAYS = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'];

export const isDay = (value: string | null | undefined): value is string =>
  typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value) && fromUtc(toUtc(value)) === value;

function toUtc(day: string) {
  const [y, m, d] = day.split('-').map(Number);
  return Date.UTC(y, m - 1, d);
}

const fromUtc = (ms: number) => new Date(ms).toISOString().slice(0, 10);

/** Today in the album's time zone (defaults to China Standard Time). */
export function today(timeZone = TIMEZONE, at = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(at);
  return parts.slice(0, 10);
}

export const addDays = (day: string, n: number) => fromUtc(toUtc(day) + n * DAY_MS);

/** Whole days from `from` to `to` (negative when `to` is earlier). */
export const daysBetween = (from: string, to: string) => Math.round((toUtc(to) - toUtc(from)) / DAY_MS);

/** "在一起的第 N 天": the day itself is day 1. */
export const dayNumber = (since: string, on: string) => daysBetween(since, on) + 1;

export const weekday = (day: string) => WEEKDAYS[new Date(toUtc(day)).getUTCDay()];

export function formatDay(day: string | null | undefined, style: 'cn' | 'dot' | 'month' = 'cn') {
  if (!day) return '';
  const [y, m, d] = day.slice(0, 10).split('-').map(Number);
  if (!y || !m) return '';
  if (style === 'dot') return `${y}.${String(m).padStart(2, '0')}.${String(d).padStart(2, '0')}`;
  if (style === 'month') return `${y}年${m}月`;
  return `${y}年${m}月${d}日`;
}

export function formatRange(start: string, end: string | null | undefined) {
  if (!end || end === start) return formatDay(start);
  const [y1, m1] = start.split('-').map(Number);
  const [y2, m2, d2] = end.split('-').map(Number);
  if (y1 !== y2) return `${formatDay(start)} – ${formatDay(end)}`;
  if (m1 !== m2) return `${formatDay(start)} – ${m2}月${d2}日`;
  return `${formatDay(start)} – ${d2}日`;
}

/** `2024-05-20T14:30:00` → `2024年5月20日 14:30` */
export function formatTaken(takenAt: string | null | undefined) {
  if (!takenAt) return '';
  const time = takenAt.slice(11, 16);
  return `${formatDay(takenAt.slice(0, 10))}${time && time !== '00:00' ? ` ${time}` : ''}`;
}

/** `MM-DD` of the days around `day`, the day itself excluded. */
export function monthDaysAround(day: string, radius: number) {
  const result: string[] = [];
  for (let i = -radius; i <= radius; i++) {
    if (i !== 0) result.push(addDays(day, i).slice(5));
  }
  return result;
}

/** How many times `MM-DD` came around between two days, both included. */
export function occurrences(from: string, to: string, monthDay: string) {
  if (daysBetween(from, to) < 0) return 0;
  let count = 0;
  for (let year = Number(from.slice(0, 4)); year <= Number(to.slice(0, 4)); year++) {
    const candidate = `${year}-${monthDay}`;
    if (isDay(candidate) && candidate >= from && candidate <= to) count++;
  }
  return count;
}

export type Milestone = { label: string; day: string; inDays: number };

const SPECIAL_DAYS = [99, 100, 199, 200, 365, 500, 520, 999, 1000, 1314, 2000, 3000, 3344, 5200, 9999, 10000];

/** The next few days worth celebrating: round day counts and anniversaries. */
export function upcomingMilestones(since: string, on: string, count = 3): Milestone[] {
  if (!isDay(since)) return [];
  const current = dayNumber(since, on);
  const found: Milestone[] = [];

  const numbers = new Set(SPECIAL_DAYS);
  for (let n = Math.ceil((current + 1) / 100) * 100; n < current + 800; n += 100) numbers.add(n);
  for (const n of numbers) {
    if (n <= current) continue;
    const day = addDays(since, n - 1);
    found.push({ label: `在一起的第 ${n} 天`, day, inDays: daysBetween(on, day) });
  }

  const startYear = Number(since.slice(0, 4));
  for (let year = Number(on.slice(0, 4)); year <= Number(on.slice(0, 4)) + 1; year++) {
    const years = year - startYear;
    const day = `${year}-${since.slice(5)}`;
    if (years > 0 && isDay(day) && day > on) {
      found.push({ label: `在一起 ${years} 周年`, day, inDays: daysBetween(on, day) });
    }
  }

  return found.sort((a, b) => a.inDays - b.inDays).slice(0, count);
}

export function yearsAgoLabel(takenYear: number, currentYear: number) {
  const years = currentYear - takenYear;
  return years === 1 ? '一年前的今天' : `${years} 年前的今天`;
}

/** Monday-first grid of a month; `null` pads the first week. */
export function monthGrid(month: string): (string | null)[] {
  const [y, m] = month.split('-').map(Number);
  const first = Date.UTC(y, m - 1, 1);
  const daysInMonth = new Date(Date.UTC(y, m, 0)).getUTCDate();
  const lead = (new Date(first).getUTCDay() + 6) % 7;
  const cells: (string | null)[] = Array.from({ length: lead }, () => null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(fromUtc(first + (d - 1) * DAY_MS));
  while (cells.length % 7 !== 0) cells.push(null);
  return cells;
}

export function shiftMonth(month: string, delta: number) {
  const [y, m] = month.split('-').map(Number);
  const date = new Date(Date.UTC(y, m - 1 + delta, 1));
  return date.toISOString().slice(0, 7);
}

export const isMonth = (value: string | null | undefined): value is string =>
  typeof value === 'string' && /^\d{4}-(0[1-9]|1[0-2])$/.test(value);

/** Epoch ms of a wall-clock time (`YYYY-MM-DD` + `HH:mm`) in a time zone. */
export function zonedToEpoch(day: string, time: string, timeZone = TIMEZONE): number {
  const [y, m, d] = day.split('-').map(Number);
  const [hh = 0, mm = 0] = (time || '00:00').split(':').map(Number);
  const guess = Date.UTC(y, m - 1, d, hh, mm);
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat('en-US', {
      timeZone,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
      .formatToParts(new Date(guess))
      .map((part) => [part.type, Number(part.value)]),
  );
  const asZoned = Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute);
  return guess - (asZoned - guess);
}
