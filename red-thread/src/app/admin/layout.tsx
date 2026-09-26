import type { Metadata } from 'next';
import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { storageWarning } from '@/lib/storage';
import { Icon } from '@/components/Icon';
import { NavLinks } from '@/components/NavLinks';

export const metadata: Metadata = { title: '整理相册', robots: { index: false, follow: false } };

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  await requireSession('/admin');
  const warning = storageWarning();
  return (
    <div className="admin">
      <aside className="admin-side">
        <Link href="/us" className="us-brand">
          <Icon name="heart" size={18} filled />
          整理相册
        </Link>
        <NavLinks
          className="admin-nav"
          items={[
            { href: '/admin/photos', label: '照片', icon: 'image' },
            { href: '/admin/upload', label: '上传', icon: 'upload' },
            { href: '/admin/moments', label: '回忆章节', icon: 'book' },
            { href: '/admin/settings', label: '我们 & 请柬', icon: 'settings' },
            { href: '/admin/guests', label: '回执与祝福', icon: 'users' },
            { href: '/admin/share', label: '分享', icon: 'share' },
          ]}
        />
        <div className="admin-side-foot">
          <Link href="/us" className="btn btn-ghost btn-sm">
            <Icon name="home" size={15} /> 回小窝
          </Link>
          <Link href="/" className="btn btn-ghost btn-sm">
            <Icon name="eye" size={15} /> 公开页
          </Link>
        </div>
      </aside>
      <main className="admin-main">
        {warning ? <p className="notice notice-red admin-warning">{warning}</p> : null}
        {children}
      </main>
    </div>
  );
}
