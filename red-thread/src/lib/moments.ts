import 'server-only';
import { now, query, queryOne } from './db';
import type { Moment, MomentKind } from './types';

export const MOMENT_KINDS: { value: MomentKind; label: string }[] = [
  { value: 'meet', label: '初遇' },
  { value: 'date', label: '约会' },
  { value: 'trip', label: '旅行' },
  { value: 'daily', label: '日常' },
  { value: 'anniversary', label: '纪念日' },
  { value: 'milestone', label: '里程碑' },
];

const KIND_VALUES = new Set(MOMENT_KINDS.map((kind) => kind.value));
export const isMomentKind = (value: unknown): value is MomentKind =>
  typeof value === 'string' && KIND_VALUES.has(value as MomentKind);

function rowToMoment(row: Record<string, unknown>): Moment {
  const text = (value: unknown) => (value === null || value === undefined ? null : String(value));
  return {
    id: String(row.id),
    title: String(row.title),
    kind: isMomentKind(row.kind) ? row.kind : 'date',
    startsOn: String(row.starts_on),
    endsOn: text(row.ends_on),
    place: text(row.place),
    story: text(row.story),
    coverPhotoId: text(row.cover_photo_id),
    visibility: row.visibility === 'public' ? 'public' : 'private',
    createdAt: String(row.created_at),
    updatedAt: String(row.updated_at),
  };
}

export async function listMoments(options: { publicOnly?: boolean; newestFirst?: boolean } = {}) {
  const rows = await query(
    `SELECT * FROM moments ${options.publicOnly ? `WHERE visibility = 'public'` : ''}
     ORDER BY starts_on ${options.newestFirst ? 'DESC' : 'ASC'}, created_at ASC`,
  );
  return rows.map(rowToMoment);
}

export async function getMoment(id: string) {
  const row = await queryOne('SELECT * FROM moments WHERE id = $1', [id]);
  return row ? rowToMoment(row) : null;
}

export async function momentPhotoCounts() {
  const rows = await query<{ moment_id: string; total: number; pub: number }>(
    `SELECT moment_id, count(*)::int AS total,
       count(*) FILTER (WHERE visibility = 'public')::int AS pub
     FROM photos WHERE moment_id IS NOT NULL GROUP BY moment_id`,
  );
  return new Map(rows.map((row) => [row.moment_id, { total: Number(row.total), public: Number(row.pub) }]));
}

export type MomentInput = Omit<Moment, 'id' | 'createdAt' | 'updatedAt'>;

export async function insertMoment(id: string, input: MomentInput) {
  const time = now();
  await query(
    `INSERT INTO moments (id, title, kind, starts_on, ends_on, place, story, cover_photo_id, visibility, created_at, updated_at)
     VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)`,
    [id, input.title, input.kind, input.startsOn, input.endsOn, input.place, input.story, input.coverPhotoId, input.visibility, time],
  );
}

export async function updateMoment(id: string, input: MomentInput) {
  await query(
    `UPDATE moments SET title=$2, kind=$3, starts_on=$4, ends_on=$5, place=$6, story=$7,
       cover_photo_id=$8, visibility=$9, updated_at=$10 WHERE id=$1`,
    [id, input.title, input.kind, input.startsOn, input.endsOn, input.place, input.story, input.coverPhotoId, input.visibility, now()],
  );
}

export async function deleteMoment(id: string) {
  await query('UPDATE photos SET moment_id = NULL WHERE moment_id = $1', [id]);
  await query('DELETE FROM moments WHERE id = $1', [id]);
}
