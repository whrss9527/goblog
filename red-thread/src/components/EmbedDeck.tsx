'use client';

import { useEffect, useState, type CSSProperties } from 'react';
import type { PhotoCard } from '@/lib/types';
import { seeded } from '@/lib/ui';
import { Polaroid } from './Polaroid';

/**
 * A little pile of polaroids that deals itself, one card every few seconds.
 * Made to sit inside someone else's page (an H5 invitation, a blog post…).
 */
export function EmbedDeck({ photos, interval = 3600 }: { photos: PhotoCard[]; interval?: number }) {
  const [order, setOrder] = useState(() => photos.map((_, i) => i));
  const [leaving, setLeaving] = useState<number | null>(null);
  const [paused, setPaused] = useState(false);

  const deal = () => {
    if (leaving !== null || order.length < 2) return;
    const top = order[0];
    setLeaving(top);
    window.setTimeout(() => {
      setOrder((current) => [...current.slice(1), current[0]]);
      setLeaving(null);
    }, 700);
  };

  useEffect(() => {
    if (paused || photos.length < 2) return;
    const timer = window.setInterval(deal, interval);
    return () => window.clearInterval(timer);
  });

  if (photos.length === 0) return null;
  const visible = order.slice(0, 5);

  return (
    <button
      type="button"
      className="deck"
      onClick={deal}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      aria-label="下一张"
    >
      {visible
        .map((photoIndex, depth) => {
          const photo = photos[photoIndex];
          return (
            <Polaroid
              key={photo.id}
              photo={photo}
              eager={depth < 2}
              tape={depth === 0}
              className={`deck-card ${leaving === photoIndex ? 'leaving' : ''}`}
              tilt={`${(seeded(photo.id) - 0.5) * (depth === 0 ? 4 : 12)}deg`}
              style={
                {
                  '--dx': `${(seeded(photo.id, 5) - 0.5) * depth * 6}px`,
                  '--dy': `${depth * 5}px`,
                  zIndex: 10 - depth,
                } as CSSProperties
              }
            />
          );
        })
        .reverse()}
    </button>
  );
}
