import { SignJWT, jwtVerify } from 'jose';
import type { Partner } from './types';

/** Shared by the proxy and server code, so it must stay free of Node-only imports. */
export const SESSION_COOKIE = 'rt_session';
export const SESSION_DAYS = 30;

const DEV_SECRET = 'red-thread-local-development-secret-not-for-production';

export function authSecret(): Uint8Array | null {
  const secret = process.env.AUTH_SECRET;
  if (secret && secret.length >= 16) return new TextEncoder().encode(secret);
  if (process.env.NODE_ENV !== 'production') return new TextEncoder().encode(DEV_SECRET);
  return null;
}

export async function signSession(partner: Partner): Promise<string> {
  const secret = authSecret();
  if (!secret) throw new Error('AUTH_SECRET is not configured');
  return new SignJWT({ p: partner })
    .setProtectedHeader({ alg: 'HS256' })
    .setIssuedAt()
    .setExpirationTime(`${SESSION_DAYS}d`)
    .sign(secret);
}

export async function verifySession(token: string | undefined): Promise<{ partner: Partner } | null> {
  const secret = authSecret();
  if (!token || !secret) return null;
  try {
    const { payload } = await jwtVerify(token, secret, { algorithms: ['HS256'] });
    return payload.p === 'a' || payload.p === 'b' ? { partner: payload.p } : null;
  } catch {
    return null;
  }
}
