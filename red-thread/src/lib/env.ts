/**
 * Read an environment variable when the request is served, not when the app
 * is built. Next.js inlines `process.env.NEXT_PUBLIC_*` at build time; a
 * dynamic lookup keeps `docker run -e NEXT_PUBLIC_DOMAIN=…` working.
 */
export const runtimeEnv = (name: string): string | undefined => process.env[name] || undefined;

/** The album's public origin, e.g. `https://love.example.com`, if configured. */
export function configuredOrigin(): string | undefined {
  const domain = runtimeEnv('NEXT_PUBLIC_DOMAIN') ?? runtimeEnv('VERCEL_PROJECT_PRODUCTION_URL');
  if (!domain) return undefined;
  return (domain.startsWith('http') ? domain : `https://${domain}`).replace(/\/+$/, '');
}
