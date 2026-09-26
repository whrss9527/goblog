import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  addDays,
  dayNumber,
  formatRange,
  isDay,
  monthGrid,
  occurrences,
  today,
  upcomingMilestones,
  weekday,
} from './dates.ts';

test('day counting treats the first day as day one', () => {
  assert.equal(dayNumber('2020-05-20', '2020-05-20'), 1);
  assert.equal(dayNumber('2020-05-20', '2021-05-20'), 366);
  assert.equal(addDays('2024-02-28', 1), '2024-02-29');
});

test('isDay rejects impossible dates', () => {
  assert.ok(isDay('2024-02-29'));
  assert.ok(!isDay('2023-02-29'));
  assert.ok(!isDay('2023-13-01'));
});

test('today follows the album time zone, not the server', () => {
  const lateUtc = new Date('2024-05-19T18:30:00Z');
  assert.equal(today('Asia/Shanghai', lateUtc), '2024-05-20');
  assert.equal(today('UTC', lateUtc), '2024-05-19');
});

test('occurrences counts each valentine once', () => {
  assert.equal(occurrences('2020-05-20', '2024-02-14', '02-14'), 4);
  assert.equal(occurrences('2020-05-20', '2024-02-13', '02-14'), 3);
  assert.equal(occurrences('2020-01-01', '2024-12-31', '02-29'), 2);
});

test('upcoming milestones are sorted and in the future', () => {
  const list = upcomingMilestones('2020-05-20', '2023-02-11');
  assert.equal(list.length, 3);
  assert.deepEqual(
    list.slice(0, 2).map((m) => [m.label, m.day]),
    [
      ['在一起的第 999 天', '2023-02-12'],
      ['在一起的第 1000 天', '2023-02-13'],
    ],
  );
  assert.ok(list.every((m, i) => m.inDays > 0 && (i === 0 || list[i - 1].inDays <= m.inDays)));
});

test('month grid starts on monday', () => {
  const grid = monthGrid('2024-05');
  assert.equal(grid.length % 7, 0);
  assert.equal(grid[2], '2024-05-01');
  assert.equal(weekday('2024-05-01'), '星期三');
});

test('ranges collapse shared parts', () => {
  assert.equal(formatRange('2024-05-01', '2024-05-05'), '2024年5月1日 – 5日');
  assert.equal(formatRange('2024-04-30', '2024-05-02'), '2024年4月30日 – 5月2日');
});

test('wedding time converts from the album time zone', async () => {
  const { zonedToEpoch } = await import('./dates.ts');
  assert.equal(new Date(zonedToEpoch('2027-05-20', '11:58', 'Asia/Shanghai')).toISOString(), '2027-05-20T03:58:00.000Z');
  assert.equal(new Date(zonedToEpoch('2027-01-10', '09:00', 'America/New_York')).toISOString(), '2027-01-10T14:00:00.000Z');
});
