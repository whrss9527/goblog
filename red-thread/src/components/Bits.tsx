'use client';

import { useRouter } from 'next/navigation';
import { useRef, useState, type CSSProperties } from 'react';
import { seeded } from '@/lib/ui';

/** Postage stamp with a round postmark; purely decorative. */
export function Stamp({ place, date, thumb }: { place: string; date: string; thumb: string | null }) {
  const id = `pm-${Math.floor(seeded(place) * 1e6)}`;
  return (
    <figure className="stamp" style={{ '--tilt': `${(seeded(place, 3) - 0.5) * 10}deg` } as CSSProperties}>
      <div className="stamp-paper">
        <div className="stamp-img" style={thumb ? { backgroundImage: `url("${thumb}")` } : undefined}>
          {thumb ? null : (
            <svg width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4">
              <path d="M12 21s-6.5-5.6-6.5-11a6.5 6.5 0 0 1 13 0c0 5.4-6.5 11-6.5 11z" />
            </svg>
          )}
        </div>
        <figcaption>
          <p className="stamp-place">{place}</p>
          <p className="stamp-date">{date}</p>
        </figcaption>
      </div>
      <svg className="postmark" viewBox="0 0 100 100" aria-hidden>
        <defs>
          <path id={id} d="M50 50 m-34 0 a34 34 0 1 1 68 0 a34 34 0 1 1 -68 0" />
        </defs>
        <circle cx="50" cy="50" r="44" fill="none" stroke="currentColor" strokeWidth="2" />
        <circle cx="50" cy="50" r="25" fill="none" stroke="currentColor" strokeWidth="1.2" />
        <text fontSize="10.5" letterSpacing="2.2" fill="currentColor" fontFamily="Cormorant Garamond, serif" fontWeight="600">
          <textPath href={`#${id}`}>LOVE POST · {date} · WITH YOU ·</textPath>
        </text>
        <path d="M50 57s-7-4.3-7-9a4 4 0 0 1 7-2.5 4 4 0 0 1 7 2.5c0 4.7-7 9-7 9z" fill="currentColor" />
      </svg>
    </figure>
  );
}

/**
 * Where the red thread ends: a little bow. Tug it five times and it takes
 * the two of us home — a door only we know about.
 */
export function Bow() {
  const router = useRouter();
  const [tugs, setTugs] = useState(0);
  const ref = useRef<HTMLButtonElement>(null);
  return (
    <button
      ref={ref}
      type="button"
      className="bow"
      aria-label="红线的尽头"
      onClick={() => {
        const el = ref.current;
        el?.classList.remove('tug');
        void el?.offsetWidth;
        el?.classList.add('tug');
        const next = tugs + 1;
        setTugs(next);
        if (next >= 5) router.push('/us');
      }}
    >
      <svg viewBox="0 0 120 80" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round">
        <path d="M60 0v26" />
        <path d="M60 30c-10-14-38-22-40-6-2 14 26 12 40 6z" />
        <path d="M60 30c10-14 38-22 40-6 2 14-26 12-40 6z" />
        <path d="M56 33c-6 10-16 26-26 40M64 33c6 10 16 26 26 40" />
        <circle cx="60" cy="31" r="4.5" fill="currentColor" />
      </svg>
    </button>
  );
}
