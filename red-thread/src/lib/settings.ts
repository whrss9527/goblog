import 'server-only';
import { cache } from 'react';
import { now, query, queryOne } from './db';
import type { Settings } from './types';

export const DEFAULT_SETTINGS: Settings = {
  siteTitle: '',
  partnerA: '阿初',
  partnerB: '小满',
  initials: 'C & M',
  togetherSince: '2020-05-20',
  firstMet: '',
  tagline: '从一个人的晴天，走到两个人的四季。',
  intro:
    '传说月下老人会用一根看不见的红线，把注定的两个人系在一起。\n这是我们那根红线上，打过的每一个结。',
  envelopeEnabled: true,
  envelopeLine: '有一封信，想亲手交给你',
  weddingEnabled: false,
  weddingDate: '',
  weddingTime: '11:58',
  weddingVenue: '',
  weddingAddress: '',
  weddingLngLat: '',
  weddingInvitation:
    '我们决定把往后的日子，都过成一起的日子。\n诚邀你来见证这个重要的时刻，分享我们的喜悦。',
  weddingSchedule: '11:18 迎宾签到\n11:58 仪式开始\n12:30 喜宴',
  dressCode: '',
  rsvpEnabled: true,
  rsvpDeadline: '',
  autoApproveNotes: false,
  music: '',
  closingLine: '余生请多指教',
};

const KEY = 'site';

export const getSettings = cache(async (): Promise<Settings> => {
  const row = await queryOne<{ value: string }>('SELECT value FROM settings WHERE key = $1', [KEY]);
  if (!row) return { ...DEFAULT_SETTINGS };
  try {
    return { ...DEFAULT_SETTINGS, ...(JSON.parse(row.value) as Partial<Settings>) };
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
});

export async function saveSettings(patch: Partial<Settings>) {
  const current = await getSettings();
  const next = { ...current, ...patch };
  await query(
    `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, $3)
     ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
    [KEY, JSON.stringify(next), now()],
  );
  return next;
}

export const siteTitle = (s: Settings) => s.siteTitle || `${s.partnerA} & ${s.partnerB} 的朝朝暮暮`;

export const partnerName = (s: Settings, partner: string | null | undefined) =>
  partner === 'a' ? s.partnerA : partner === 'b' ? s.partnerB : null;
