import type { Metadata } from 'next';
import Link from 'next/link';
import { requireSession } from '@/lib/auth';
import { getSettings, partnerName } from '@/lib/settings';
import { logout } from '@/app/login/actions';
import { Icon } from '@/components/Icon';
import { LightboxProvider } from '@/components/Lightbox';
import { NavLinks } from '@/components/NavLinks';
import { RevealObserver } from '@/components/Ambient';

export const metadata: Metadata = { title: '我们的小窝', robots: { index: false, follow: false } };

export default async function UsLayout({ children }: { children: React.ReactNode }) {
  const session = await requireSession('/us');
  const settings = await getSettings();
  return (
    <LightboxProvider>
      <RevealObserver />
      <div className="us">
        <header className="us-bar">
          <Link href="/us" className="us-brand">
            <Icon name="heart" size={18} filled />
            我们的小窝
          </Link>
          <NavLinks
            className="us-nav"
            items={[
              { href: '/us', label: '今天', icon: 'home', exact: true },
              { href: '/us/calendar', label: '日历', icon: 'calendar' },
              { href: '/us/moments', label: '回忆', icon: 'book' },
              { href: '/us/photos', label: '照片', icon: 'image' },
              { href: '/admin', label: '整理', icon: 'settings' },
            ]}
          />
          <div className="us-me">
            <Link href="/" className="btn btn-ghost btn-sm" title="看看客人眼中的相册">
              <Icon name="eye" size={16} />
              <span className="hide-sm">公开页</span>
            </Link>
            <form action={logout}>
              <button type="submit" className="btn btn-ghost btn-sm" title={`${partnerName(settings, session.partner)}，要走了吗？`}>
                <Icon name="logout" size={16} />
                <span className="hide-sm">出门</span>
              </button>
            </form>
          </div>
        </header>
        <main className="us-main">{children}</main>
      </div>
    </LightboxProvider>
  );
}
