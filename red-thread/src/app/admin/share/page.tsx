import { headers } from 'next/headers';
import { configuredOrigin } from '@/lib/env';
import { requireSession } from '@/lib/auth';
import { getSettings } from '@/lib/settings';
import { weddingReady } from '@/lib/wedding';
import { SharePanel } from '@/components/admin/SharePanel';

export const dynamic = 'force-dynamic';

async function baseUrl() {
  const origin = configuredOrigin();
  if (origin) return origin;
  const head = await headers();
  const host = head.get('x-forwarded-host') ?? head.get('host') ?? 'localhost:3000';
  const proto = head.get('x-forwarded-proto') ?? (host.startsWith('localhost') ? 'http' : 'https');
  return `${proto}://${host}`;
}

export default async function SharePage() {
  await requireSession('/admin/share');
  const [base, settings] = await Promise.all([baseUrl(), getSettings()]);
  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>分享</h1>
          <p className="muted small">
            客人只能看到“公开”的照片和回忆。
            {configuredOrigin() ? '' : '配置 NEXT_PUBLIC_DOMAIN 环境变量后，这里会使用你的正式域名。'}
          </p>
        </div>
      </header>
      <SharePanel base={base} hasWedding={weddingReady(settings)} />
    </div>
  );
}
