import { formatDay, weekday, zonedToEpoch, isDay } from './dates';
import type { Settings } from './types';

export const weddingReady = (s: Settings) => s.weddingEnabled && isDay(s.weddingDate);

/** Map apps people in China actually use, plus Apple Maps. */
export function mapLinks(s: Settings) {
  const name = s.weddingVenue || s.weddingAddress;
  const query = [s.weddingAddress, s.weddingVenue].filter(Boolean).join(' ');
  if (!query) return [];
  const [lng, lat] = s.weddingLngLat.split(',').map((v) => Number(v.trim()));
  const hasPoint = Number.isFinite(lng) && Number.isFinite(lat) && s.weddingLngLat.includes(',');
  const q = encodeURIComponent;
  return [
    {
      label: '高德地图',
      href: hasPoint
        ? `https://uri.amap.com/marker?position=${lng},${lat}&name=${q(name)}&callnative=1`
        : `https://uri.amap.com/search?keyword=${q(query)}&callnative=1`,
    },
    {
      label: '百度地图',
      href: `https://api.map.baidu.com/geocoder?address=${q(query)}&output=html&src=red-thread`,
    },
    {
      label: 'Apple 地图',
      href: hasPoint
        ? `https://maps.apple.com/?ll=${lat},${lng}&q=${q(name)}`
        : `https://maps.apple.com/?q=${q(query)}`,
    },
  ];
}

export function weddingSchedule(s: Settings) {
  return s.weddingSchedule
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const match = line.match(/^(\d{1,2}[:：]\d{2})\s*(.*)$/);
      return match ? { time: match[1].replace('：', ':'), what: match[2] } : { time: '', what: line };
    });
}

export function weddingDateParts(s: Settings) {
  const [y, m, d] = s.weddingDate.split('-').map(Number);
  return {
    year: y,
    month: m,
    day: d,
    weekday: weekday(s.weddingDate),
    dot: formatDay(s.weddingDate, 'dot').replace(/\./g, ' · '),
    cn: formatDay(s.weddingDate),
  };
}

const icsText = (value: string) => value.replace(/\\/g, '\\\\').replace(/\n/g, '\\n').replace(/[,;]/g, (c) => `\\${c}`);
const icsTime = (ms: number) => new Date(ms).toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '');

export function weddingIcs(s: Settings, url: string) {
  const start = zonedToEpoch(s.weddingDate, s.weddingTime || '12:00');
  const end = start + 4 * 3600_000;
  const lines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//red-thread//wedding//CN',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    'BEGIN:VEVENT',
    `UID:wedding-${s.weddingDate}@red-thread`,
    `DTSTAMP:${icsTime(Date.now())}`,
    `DTSTART:${icsTime(start)}`,
    `DTEND:${icsTime(end)}`,
    `SUMMARY:${icsText(`${s.partnerA} & ${s.partnerB} 的婚礼`)}`,
    `LOCATION:${icsText([s.weddingVenue, s.weddingAddress].filter(Boolean).join(' '))}`,
    `DESCRIPTION:${icsText(`${s.weddingInvitation}\n\n${url}`)}`,
    `URL:${url}`,
    'BEGIN:VALARM',
    'TRIGGER:-P1D',
    'ACTION:DISPLAY',
    `DESCRIPTION:${icsText('明天就是婚礼啦')}`,
    'END:VALARM',
    'END:VEVENT',
    'END:VCALENDAR',
  ];
  return lines.join('\r\n') + '\r\n';
}
