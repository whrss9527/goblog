'use server';

import { headers } from 'next/headers';
import { redirect } from 'next/navigation';
import { authProblem, endSession, matchAccount, startSession } from '@/lib/auth';

export type LoginState = { error?: string; email?: string };

// Per-instance brake on password guessing; good enough for a two-person site.
const failures = new Map<string, { count: number; until: number }>();

const safeNext = (value: FormDataEntryValue | null) => {
  const next = String(value ?? '');
  return next.startsWith('/') && !next.startsWith('//') ? next : '/us';
};

export async function login(_prev: LoginState, form: FormData): Promise<LoginState> {
  const email = String(form.get('email') ?? '').slice(0, 200);
  const problem = authProblem();
  if (problem) return { error: problem, email };

  const ip = ((await headers()).get('x-forwarded-for') ?? 'local').split(',')[0].trim();
  const record = failures.get(ip);
  if (record && record.until > Date.now()) {
    return { error: '试错太多次啦，歇一分钟再来。', email };
  }

  const partner = matchAccount(email, String(form.get('password') ?? ''));
  if (!partner) {
    const count = (record?.count ?? 0) + 1;
    failures.set(ip, { count, until: count >= 5 ? Date.now() + 60_000 : 0 });
    await new Promise((resolve) => setTimeout(resolve, 500));
    return { error: '暗号不对哦，再想想？', email };
  }

  failures.delete(ip);
  await startSession(partner);
  redirect(safeNext(form.get('next')));
}

export async function logout() {
  await endSession();
  redirect('/');
}
