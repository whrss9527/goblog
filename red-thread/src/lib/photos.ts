import 'server-only';
import { now, query, queryOne } from './db';
import { urlFor } from './storage';
import { partnerName } from './settings';
import type { Photo, PhotoCard, Settings, Visibility } from './types';

type Row = Record<string, unknown>;

const str = (value: unknown) => (value === null || value === undefined ? null : String(value));
const num = (value: unknown) => (value === null || value === undefined ? null : Number(value));

export function rowToPhoto(row: Row): Photo {
  return {
    id: String(row.id),
    lgKey: String(row.lg_key),
    smKey: String(row.sm_key),
    origKey: str(row.orig_key),
    width: Number(row.width),
    height: Number(row.height),
    blurData: str(row.blur_data),
    color: str(row.color),
    caption: str(row.caption),
    note: str(row.note),
    takenAt: str(row.taken_at),
    place: str(row.place),
    latitude: num(row.latitude),
    longitude: num(row.longitude),
    camera: str(row.camera),
    author: row.author === 'a' || row.author === 'b' ? row.author : null,
    momentId: str(row.moment_id),
    visibility: row.visibility === 'public' ? 'public' : 'private',
    featured: Boolean(row.featured),
    favorite: Boolean(row.favorite),
    createdAt: String(row.created_at),
    updatedAt: String(row.updated_at),
  };
}

/** Taken time when we know it, otherwise upload time. Sorts as text. */
const WHEN = `COALESCE(taken_at, substr(created_at, 1, 19))`;

export type PhotoFilter = {
  visibility?: Visibility;
  momentId?: string;
  unassigned?: boolean;
  featured?: boolean;
  favorite?: boolean;
  order?: 'oldest' | 'newest' | 'uploaded';
  limit?: number;
  offset?: number;
};

export async function listPhotos(filter: PhotoFilter = {}): Promise<Photo[]> {
  const where: string[] = [];
  const params: unknown[] = [];
  const add = (clause: string, value: unknown) => {
    params.push(value);
    where.push(clause.replace('?', `$${params.length}`));
  };
  if (filter.visibility) add('visibility = ?', filter.visibility);
  if (filter.momentId) add('moment_id = ?', filter.momentId);
  if (filter.unassigned) where.push('moment_id IS NULL');
  if (filter.featured) where.push('featured = TRUE');
  if (filter.favorite) where.push('favorite = TRUE');

  const order =
    filter.order === 'newest'
      ? `${WHEN} DESC, id`
      : filter.order === 'uploaded'
        ? 'created_at DESC, id'
        : `${WHEN} ASC, id`;
  let sql = `SELECT * FROM photos ${where.length ? `WHERE ${where.join(' AND ')}` : ''} ORDER BY ${order}`;
  if (filter.limit) sql += ` LIMIT ${Math.max(1, Math.floor(filter.limit))}`;
  if (filter.offset) sql += ` OFFSET ${Math.max(0, Math.floor(filter.offset))}`;
  return (await query(sql, params)).map(rowToPhoto);
}

export async function countPhotos() {
  const rows = await query<{ visibility: string; n: number }>(
    'SELECT visibility, count(*)::int AS n FROM photos GROUP BY visibility',
  );
  const total = rows.reduce((sum, row) => sum + Number(row.n), 0);
  const pub = Number(rows.find((row) => row.visibility === 'public')?.n ?? 0);
  return { total, public: pub, private: total - pub };
}

export async function getPhoto(id: string) {
  const row = await queryOne('SELECT * FROM photos WHERE id = $1', [id]);
  return row ? rowToPhoto(row) : null;
}

export async function getPhotos(ids: string[]) {
  if (ids.length === 0) return [];
  const marks = ids.map((_, i) => `$${i + 1}`).join(',');
  return (await query(`SELECT * FROM photos WHERE id IN (${marks})`, ids)).map(rowToPhoto);
}

export async function findPhotoByKey(key: string) {
  const row = await queryOne(
    'SELECT * FROM photos WHERE lg_key = $1 OR sm_key = $1 OR orig_key = $1 LIMIT 1',
    [key],
  );
  return row ? rowToPhoto(row) : null;
}

/** Photos taken on these `MM-DD` days in years before `beforeYear`. */
export async function photosOnDays(monthDays: string[], beforeYear: number) {
  if (monthDays.length === 0) return [];
  const marks = monthDays.map((_, i) => `$${i + 2}`).join(',');
  const rows = await query(
    `SELECT * FROM photos
     WHERE taken_at IS NOT NULL AND substr(taken_at, 1, 4) < $1 AND substr(taken_at, 6, 5) IN (${marks})
     ORDER BY taken_at DESC`,
    [String(beforeYear), ...monthDays],
  );
  return rows.map(rowToPhoto);
}

export async function photosInMonth(month: string) {
  const rows = await query(
    `SELECT * FROM photos WHERE substr(taken_at, 1, 7) = $1 ORDER BY taken_at ASC`,
    [month],
  );
  return rows.map(rowToPhoto);
}

export async function monthsWithPhotos() {
  return query<{ month: string; n: number }>(
    `SELECT substr(taken_at, 1, 7) AS month, count(*)::int AS n FROM photos
     WHERE taken_at IS NOT NULL GROUP BY 1 ORDER BY 1 DESC`,
  );
}

export async function randomPhoto(visibility?: Visibility) {
  const row = await queryOne(
    `SELECT * FROM photos ${visibility ? 'WHERE visibility = $1' : ''} ORDER BY random() LIMIT 1`,
    visibility ? [visibility] : [],
  );
  return row ? rowToPhoto(row) : null;
}

/** The earliest photo of every moment, used as a fallback cover. */
export async function firstPhotoPerMoment() {
  const rows = await query(
    `SELECT DISTINCT ON (moment_id) * FROM photos WHERE moment_id IS NOT NULL
     ORDER BY moment_id, ${WHEN} ASC, id`,
  );
  return new Map(rows.map((row) => [String(row.moment_id), rowToPhoto(row)]));
}

export async function privateCountsByMoment() {
  const rows = await query<{ moment_id: string; n: number }>(
    `SELECT moment_id, count(*)::int AS n FROM photos
     WHERE visibility = 'private' AND moment_id IS NOT NULL GROUP BY moment_id`,
  );
  return new Map(rows.map((row) => [row.moment_id, Number(row.n)]));
}

export type NewPhoto = Omit<Photo, 'createdAt' | 'updatedAt' | 'favorite' | 'featured'> &
  Partial<Pick<Photo, 'favorite' | 'featured'>>;

export async function insertPhoto(photo: NewPhoto) {
  const time = now();
  await query(
    `INSERT INTO photos (id, lg_key, sm_key, orig_key, width, height, blur_data, color, caption, note,
       taken_at, place, latitude, longitude, camera, author, moment_id, visibility, featured, favorite,
       created_at, updated_at)
     VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$21)`,
    [
      photo.id, photo.lgKey, photo.smKey, photo.origKey, photo.width, photo.height, photo.blurData,
      photo.color, photo.caption, photo.note, photo.takenAt, photo.place, photo.latitude,
      photo.longitude, photo.camera, photo.author, photo.momentId, photo.visibility,
      photo.featured ?? false, photo.favorite ?? false, time,
    ],
  );
}

const EDITABLE: Record<string, string> = {
  caption: 'caption',
  note: 'note',
  takenAt: 'taken_at',
  place: 'place',
  author: 'author',
  momentId: 'moment_id',
  visibility: 'visibility',
  featured: 'featured',
  favorite: 'favorite',
};

export type PhotoPatch = Partial<
  Pick<Photo, 'caption' | 'note' | 'takenAt' | 'place' | 'author' | 'momentId' | 'visibility' | 'featured' | 'favorite'>
>;

export async function updatePhotos(ids: string[], patch: PhotoPatch) {
  const entries = Object.entries(patch).filter(([field, value]) => EDITABLE[field] && value !== undefined);
  if (ids.length === 0 || entries.length === 0) return;
  const params: unknown[] = entries.map(([, value]) => value);
  const sets = entries.map(([field], i) => `${EDITABLE[field]} = $${i + 1}`);
  params.push(now());
  sets.push(`updated_at = $${params.length}`);
  const marks = ids.map((id) => {
    params.push(id);
    return `$${params.length}`;
  });
  await query(`UPDATE photos SET ${sets.join(', ')} WHERE id IN (${marks.join(',')})`, params);
}

export async function deletePhotos(ids: string[]) {
  if (ids.length === 0) return [];
  const photos = await getPhotos(ids);
  const marks = ids.map((_, i) => `$${i + 1}`).join(',');
  await query(`DELETE FROM photos WHERE id IN (${marks})`, ids);
  await query(`UPDATE moments SET cover_photo_id = NULL WHERE cover_photo_id IN (${marks})`, ids);
  return photos;
}

/**
 * Turn rows into what the browser may see. GPS, camera and the original file
 * stay on the server.
 */
export async function toCards(
  photos: Photo[],
  settings: Settings,
  options: { forUs?: boolean } = {},
): Promise<PhotoCard[]> {
  return Promise.all(
    photos.map(async (photo) => {
      const isPrivate = photo.visibility === 'private';
      const [src, thumb] = await Promise.all([
        urlFor(photo.lgKey, isPrivate),
        urlFor(photo.smKey, isPrivate),
      ]);
      const card: PhotoCard = {
        id: photo.id,
        src,
        thumb,
        width: photo.width,
        height: photo.height,
        blurData: photo.blurData,
        color: photo.color,
        caption: photo.caption,
        note: photo.note,
        takenAt: photo.takenAt,
        place: photo.place,
        author: partnerName(settings, photo.author),
      };
      if (options.forUs) {
        card.isPrivate = isPrivate;
        card.favorite = photo.favorite;
      }
      return card;
    }),
  );
}
