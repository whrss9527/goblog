import 'server-only';
import fs from 'node:fs';
import path from 'node:path';
import { SCHEMA } from './schema';

/**
 * Two drivers, one dialect:
 * - `POSTGRES_URL` set (Vercel / Neon / Supabase / your own server) → node-postgres
 * - otherwise → PGlite, a real Postgres compiled to WASM that keeps its data
 *   in `.data/pglite`. Zero setup for local development and small self-hosts.
 *
 * All timestamps are stored as ISO text so both drivers return the same values.
 */
type Driver = {
  query: (text: string, params?: unknown[]) => Promise<Record<string, unknown>[]>;
};

const globalForDb = globalThis as unknown as { __redThreadDb?: Promise<Driver> };

async function createDriver(): Promise<Driver> {
  const url = process.env.POSTGRES_URL;
  if (url) {
    const { Pool } = await import('pg');
    const pool = new Pool({
      // node-postgres treats `sslmode` in the URL differently from `ssl` below.
      connectionString: url.replace(/([?&])sslmode=[^&]*&?/, '$1').replace(/[?&]$/, ''),
      ssl: process.env.DISABLE_POSTGRES_SSL === '1' ? false : true,
      max: 5,
    });
    return {
      query: async (text, params = []) => (await pool.query(text, params)).rows,
    };
  }

  const { PGlite } = await import('@electric-sql/pglite');
  const dir = process.env.PGLITE_DIR || path.join(/*turbopackIgnore: true*/ process.cwd(), '.data', 'pglite');
  fs.mkdirSync(path.dirname(dir), { recursive: true });
  const db = new PGlite(dir);
  await db.waitReady;
  return {
    query: async (text, params = []) =>
      (await db.query<Record<string, unknown>>(text, params)).rows,
  };
}

async function init(): Promise<Driver> {
  const driver = await createDriver();
  for (const statement of SCHEMA) {
    try {
      await driver.query(statement);
    } catch (error) {
      // Two cold starts racing on CREATE ... IF NOT EXISTS can collide on the
      // catalog; the other one won, which is all we need.
      const code = (error as { code?: string }).code;
      if (code !== '23505' && code !== '42P07' && code !== '42710') throw error;
    }
  }
  return driver;
}

function driver(): Promise<Driver> {
  if (!globalForDb.__redThreadDb) {
    globalForDb.__redThreadDb = init().catch((error) => {
      globalForDb.__redThreadDb = undefined;
      throw error;
    });
  }
  return globalForDb.__redThreadDb;
}

export async function query<T = Record<string, unknown>>(
  text: string,
  params: unknown[] = [],
): Promise<T[]> {
  const db = await driver();
  return (await db.query(text, params)) as T[];
}

export async function queryOne<T = Record<string, unknown>>(
  text: string,
  params: unknown[] = [],
): Promise<T | null> {
  const rows = await query<T>(text, params);
  return rows[0] ?? null;
}

export const now = () => new Date().toISOString();
