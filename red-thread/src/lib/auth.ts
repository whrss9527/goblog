import 'server-only';
import { createHash, timingSafeEqual } from 'node:crypto';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import { SESSION_COOKIE, SESSION_DAYS, authSecret, signSession, verifySession } from './session';
import type { Partner } from './types';

type Account = { partner: Partner; email: string; password: string };

export const DEV_ACCOUNT = { email: 'us@love.local', password: 'together' };
const isProd = process.env.NODE_ENV === 'production';

function accounts(): Account[] {
  const list: Account[] = [];
  const { ADMIN_EMAIL, ADMIN_PASSWORD, PARTNER_EMAIL, PARTNER_PASSWORD } = process.env;
  if (ADMIN_EMAIL && ADMIN_PASSWORD) {
    list.push({ partner: 'a', email: ADMIN_EMAIL, password: ADMIN_PASSWORD });
  }
  if (PARTNER_EMAIL && PARTNER_PASSWORD) {
    list.push({ partner: 'b', email: PARTNER_EMAIL, password: PARTNER_PASSWORD });
  }
  if (list.length === 0 && !isProd) list.push({ partner: 'a', ...DEV_ACCOUNT });
  return list;
}

/** Why nobody can sign in yet, in words the two of us understand. */
export function authProblem(): string | null {
  if (!authSecret()) return '还没有配置 AUTH_SECRET 环境变量，请先设置一个足够长的随机字符串。';
  if (accounts().length === 0) return '还没有配置 ADMIN_EMAIL / ADMIN_PASSWORD 环境变量。';
  return null;
}

export const usingDevAccount = () =>
  !isProd && !process.env.ADMIN_EMAIL && !process.env.PARTNER_EMAIL;

const digest = (value: string) => createHash('sha256').update(value).digest();
const same = (a: string, b: string) => timingSafeEqual(digest(a), digest(b));

/** Checks every account without returning early, so timing reveals nothing. */
export function matchAccount(email: string, password: string): Partner | null {
  let found: Partner | null = null;
  for (const account of accounts()) {
    const emailOk = same(email.trim().toLowerCase(), account.email.trim().toLowerCase());
    const passwordOk = same(password, account.password);
    if (emailOk && passwordOk && !found) found = account.partner;
  }
  return found;
}

export async function getSession(): Promise<{ partner: Partner } | null> {
  const jar = await cookies();
  return verifySession(jar.get(SESSION_COOKIE)?.value);
}

/** For pages: bounce to the login page when signed out. */
export async function requireSession(next = '/us'): Promise<{ partner: Partner }> {
  const session = await getSession();
  if (!session) redirect(`/login?next=${encodeURIComponent(next)}`);
  return session;
}

/** For server actions and route handlers: never trust the proxy alone. */
export async function assertSession(): Promise<{ partner: Partner }> {
  const session = await getSession();
  if (!session) throw new Error('需要先登录');
  return session;
}

export async function startSession(partner: Partner) {
  const jar = await cookies();
  jar.set(SESSION_COOKIE, await signSession(partner), {
    httpOnly: true,
    secure: isProd,
    sameSite: 'lax',
    path: '/',
    maxAge: SESSION_DAYS * 24 * 60 * 60,
  });
}

export async function endSession() {
  const jar = await cookies();
  jar.delete(SESSION_COOKIE);
}
