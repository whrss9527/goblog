/**
 * Idempotent schema, applied on the first query of every server process.
 * New columns go in as `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` at the end,
 * so existing albums upgrade themselves on the next deploy.
 */
export const SCHEMA: string[] = [
  `CREATE TABLE IF NOT EXISTS moments (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'date',
    starts_on TEXT NOT NULL,
    ends_on TEXT,
    place TEXT,
    story TEXT,
    cover_photo_id TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
  )`,
  `CREATE TABLE IF NOT EXISTS photos (
    id TEXT PRIMARY KEY,
    lg_key TEXT NOT NULL,
    sm_key TEXT NOT NULL,
    orig_key TEXT,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    blur_data TEXT,
    color TEXT,
    caption TEXT,
    note TEXT,
    taken_at TEXT,
    place TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    camera TEXT,
    author TEXT,
    moment_id TEXT REFERENCES moments(id) ON DELETE SET NULL,
    visibility TEXT NOT NULL DEFAULT 'private',
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    favorite BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
  )`,
  `CREATE INDEX IF NOT EXISTS photos_taken_at_idx ON photos (taken_at)`,
  `CREATE INDEX IF NOT EXISTS photos_moment_idx ON photos (moment_id)`,
  `CREATE INDEX IF NOT EXISTS photos_visibility_idx ON photos (visibility)`,
  `CREATE TABLE IF NOT EXISTS guest_notes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    message TEXT,
    attending TEXT,
    party_size INTEGER,
    contact TEXT,
    approved BOOLEAN NOT NULL DEFAULT FALSE,
    ip_hash TEXT,
    created_at TEXT NOT NULL
  )`,
  `CREATE INDEX IF NOT EXISTS guest_notes_ip_idx ON guest_notes (ip_hash, created_at)`,
  `CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
  )`,
];
