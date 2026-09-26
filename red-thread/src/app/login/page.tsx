import type { Metadata } from 'next';
import Link from 'next/link';
import { redirect } from 'next/navigation';
import { DEV_ACCOUNT, getSession, usingDevAccount } from '@/lib/auth';
import { LoginForm } from './LoginForm';

export const metadata: Metadata = { title: '我们的入口', robots: { index: false } };

type Props = { searchParams: Promise<{ next?: string }> };

export default async function LoginPage({ searchParams }: Props) {
  const { next } = await searchParams;
  const target = next && next.startsWith('/') && !next.startsWith('//') ? next : '/us';
  if (await getSession()) redirect(target);
  const hint = usingDevAccount()
    ? `本地开发模式：用 ${DEV_ACCOUNT.email} / ${DEV_ACCOUNT.password} 登录。上线前请配置 ADMIN_EMAIL 和 ADMIN_PASSWORD。`
    : null;
  return (
    <main className="login">
      <div className="login-card">
        <p className="section-kicker">Welcome home</p>
        <h1 className="login-title">只属于我们的入口</h1>
        <p className="muted">这里有公开相册之外，那些只给彼此看的照片。</p>
        <LoginForm next={target} hint={hint} />
      </div>
      <Link href="/" className="login-back">
        ← 回到公开的相册
      </Link>
    </main>
  );
}
