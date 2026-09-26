import type { Metadata, Viewport } from 'next';
import { configuredOrigin } from '@/lib/env';
import '@fontsource/noto-serif-sc/400.css';
import '@fontsource/noto-serif-sc/600.css';
import '@fontsource/long-cang/400.css';
import '@fontsource/cormorant-garamond/400.css';
import '@fontsource/cormorant-garamond/600.css';
import '@fontsource/great-vibes/400.css';
import '@/styles/base.css';
import '@/styles/lightbox.css';
import '@/styles/public.css';
import '@/styles/us.css';
import '@/styles/admin.css';

export function generateMetadata(): Metadata {
  const origin = configuredOrigin();
  return {
    metadataBase: origin ? new URL(origin) : undefined,
    title: { default: '红线 · 我们的相册', template: '%s' },
    description: '一本会讲故事的恋爱相册。',
    icons: { icon: '/icon.svg' },
    formatDetection: { telephone: false },
  };
}

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  viewportFit: 'cover',
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#fbf6ee' },
    { media: '(prefers-color-scheme: dark)', color: '#1d1719' },
  ],
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>
        {children}
        <noscript>
          <style>{`[data-reveal]{opacity:1;transform:none}.polaroid-img img,.tile img{opacity:1}.env-overlay{display:none}`}</style>
        </noscript>
      </body>
    </html>
  );
}
