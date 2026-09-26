import type { NextConfig } from 'next';

// Pages only the two of us should see must never be framed by another site.
const privateFrameHeaders = [
  { key: 'Content-Security-Policy', value: "frame-ancestors 'self'" },
  { key: 'X-Frame-Options', value: 'SAMEORIGIN' },
];

const nextConfig: NextConfig = {
  // `NEXT_OUTPUT=standalone npm run build` produces the self-contained server
  // used by the Dockerfile; Vercel ignores this and builds as usual.
  output: process.env.NEXT_OUTPUT === 'standalone' ? 'standalone' : undefined,
  poweredByHeader: false,
  devIndicators: false,
  serverExternalPackages: ['@electric-sql/pglite', 'pg'],
  async headers() {
    return [
      { source: '/admin/:path*', headers: privateFrameHeaders },
      { source: '/us/:path*', headers: privateFrameHeaders },
      { source: '/us', headers: privateFrameHeaders },
      { source: '/login', headers: privateFrameHeaders },
      // The embed view exists to be placed inside wedding invitation pages.
      {
        source: '/embed',
        headers: [{ key: 'Content-Security-Policy', value: 'frame-ancestors *' }],
      },
      {
        source: '/demo/:path*',
        headers: [{ key: 'Cache-Control', value: 'public, max-age=86400' }],
      },
    ];
  },
};

export default nextConfig;
