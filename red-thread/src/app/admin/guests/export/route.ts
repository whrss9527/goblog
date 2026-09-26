import { getSession } from '@/lib/auth';
import { listNotes } from '@/lib/guests';

const cell = (value: unknown) => {
  const text = String(value ?? '');
  // Leading =,+,-,@ would be read as a formula by spreadsheet apps.
  const safe = /^[=+\-@]/.test(text) ? `'${text}` : text;
  return `"${safe.replace(/"/g, '""')}"`;
};

export async function GET() {
  if (!(await getSession())) return new Response('Not found', { status: 404 });
  const notes = await listNotes();
  const header = ['名字', '是否出席', '人数', '联系方式', '祝福', '已上墙', '时间'];
  const attending: Record<string, string> = { yes: '到场', maybe: '不确定', no: '缺席' };
  const lines = notes.map((n) =>
    [n.name, n.attending ? attending[n.attending] : '', n.partySize ?? '', n.contact, n.message, n.approved ? '是' : '否', n.createdAt]
      .map(cell)
      .join(','),
  );
  // BOM so Excel opens the Chinese text correctly.
  const body = '﻿' + [header.map(cell).join(','), ...lines].join('\r\n');
  return new Response(body, {
    headers: {
      'Content-Type': 'text/csv; charset=utf-8',
      'Content-Disposition': 'attachment; filename="guests.csv"',
      'Cache-Control': 'no-store',
    },
  });
}
