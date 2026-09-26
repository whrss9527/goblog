import 'server-only';
import { now, query, queryOne } from './db';
import type { Attendance, GuestNote } from './types';

function rowToNote(row: Record<string, unknown>): GuestNote {
  const text = (value: unknown) => (value === null || value === undefined ? null : String(value));
  const attending = row.attending;
  return {
    id: String(row.id),
    name: String(row.name),
    message: text(row.message),
    attending: attending === 'yes' || attending === 'no' || attending === 'maybe' ? attending : null,
    partySize: row.party_size === null || row.party_size === undefined ? null : Number(row.party_size),
    contact: text(row.contact),
    approved: Boolean(row.approved),
    createdAt: String(row.created_at),
  };
}

export async function insertNote(note: {
  id: string;
  name: string;
  message: string | null;
  attending: Attendance | null;
  partySize: number | null;
  contact: string | null;
  approved: boolean;
  ipHash: string;
}) {
  await query(
    `INSERT INTO guest_notes (id, name, message, attending, party_size, contact, approved, ip_hash, created_at)
     VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
    [note.id, note.name, note.message, note.attending, note.partySize, note.contact, note.approved, note.ipHash, now()],
  );
}

export async function recentNotesFrom(ipHash: string, minutes: number) {
  const since = new Date(Date.now() - minutes * 60_000).toISOString();
  const row = await queryOne<{ n: number }>(
    'SELECT count(*)::int AS n FROM guest_notes WHERE ip_hash = $1 AND created_at > $2',
    [ipHash, since],
  );
  return Number(row?.n ?? 0);
}

/** Blessings guests may read: approved, and actually containing words. */
export async function publicBlessings(limit = 60) {
  const rows = await query(
    `SELECT * FROM guest_notes WHERE approved = TRUE AND message IS NOT NULL AND message <> ''
     ORDER BY created_at DESC LIMIT ${Math.floor(limit)}`,
  );
  return rows.map(rowToNote);
}

export async function listNotes() {
  return (await query('SELECT * FROM guest_notes ORDER BY created_at DESC')).map(rowToNote);
}

export async function setNoteApproved(id: string, approved: boolean) {
  await query('UPDATE guest_notes SET approved = $2 WHERE id = $1', [id, approved]);
}

export async function deleteNote(id: string) {
  await query('DELETE FROM guest_notes WHERE id = $1', [id]);
}
