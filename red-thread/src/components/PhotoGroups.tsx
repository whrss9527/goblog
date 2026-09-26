'use client';

import type { CSSProperties } from 'react';
import type { PhotoCard } from '@/lib/types';
import { seeded } from '@/lib/ui';
import { FadeImg, Polaroid, blurStyle } from './Polaroid';
import { useLightbox } from './Lightbox';
import { Icon } from './Icon';

/** Fanned stack of our favourite photos at the top of the page. */
export function HeroFan({ photos }: { photos: PhotoCard[] }) {
  const { open } = useLightbox();
  const shown = photos.slice(0, 5);
  const mid = (shown.length - 1) / 2;
  return (
    <div className={`fan fan-${shown.length}`}>
      {shown.map((photo, i) => {
        const offset = i - mid;
        return (
          <Polaroid
            key={photo.id}
            photo={photo}
            eager
            tape={i === Math.round(mid)}
            className="fan-card"
            tilt={`${offset * 7 + (seeded(photo.id) - 0.5) * 3}deg`}
            style={
              {
                '--i': offset,
                '--y': `${Math.abs(offset) * 7}%`,
                zIndex: 10 - Math.abs(Math.round(offset)),
              } as CSSProperties
            }
            onOpen={() => open(shown, i)}
          />
        );
      })}
    </div>
  );
}

/** Up to three polaroids tossed on the table, plus how many more there are. */
export function PolaroidCluster({ photos, label }: { photos: PhotoCard[]; label: string }) {
  const { open } = useLightbox();
  const shown = photos.slice(0, 3);
  const more = photos.length - shown.length;
  return (
    <div className={`cluster cluster-${shown.length}`}>
      {shown.map((photo, i) => (
        <Polaroid key={photo.id} photo={photo} className="cluster-card" onOpen={() => open(photos, i)} />
      ))}
      {more > 0 ? (
        <button type="button" className="cluster-more" onClick={() => open(photos, shown.length)}>
          <Icon name="image" size={16} /> 还有 {more} 张
          <span className="sr-only">{label}</span>
        </button>
      ) : null}
    </div>
  );
}

/** A corkboard of photos that don't belong to any chapter yet. */
export function Corkboard({ photos }: { photos: PhotoCard[] }) {
  const { open } = useLightbox();
  return (
    <div className="cork">
      {photos.map((photo, i) => (
        <Polaroid key={photo.id} photo={photo} className="cork-card" onOpen={() => open(photos, i)} />
      ))}
    </div>
  );
}

/** A horizontal strip of polaroids (那年今日). */
export function PolaroidStrip({ photos }: { photos: PhotoCard[] }) {
  const { open } = useLightbox();
  return (
    <div className="strip">
      {photos.map((photo, i) => (
        <Polaroid key={photo.id} photo={photo} showLock className="strip-card" onOpen={() => open(photos, i)} />
      ))}
    </div>
  );
}

/** Plain masonry used in the private area, lock badges included. */
export function PhotoMasonry({ photos }: { photos: PhotoCard[] }) {
  const { open } = useLightbox();
  return (
    <div className="masonry">
      {photos.map((photo, i) => (
        <button
          type="button"
          key={photo.id}
          className="tile"
          style={{ ...blurStyle(photo), aspectRatio: `${photo.width} / ${photo.height}` }}
          onClick={() => open(photos, i)}
          aria-label={photo.caption || '打开照片'}
        >
          <FadeImg src={photo.thumb} alt={photo.caption ?? ''} width={photo.width} height={photo.height} />
          {photo.isPrivate ? (
            <span className="lock-badge" title="只有我们看得到">
              <Icon name="lock" size={13} />
            </span>
          ) : null}
          {photo.favorite ? (
            <span className="tile-heart">
              <Icon name="heart" filled size={18} />
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}

/** Opens one photo (used for "抽一张回忆"). */
export function OpenPhotoButton({
  photos,
  index = 0,
  className,
  children,
}: {
  photos: PhotoCard[];
  index?: number;
  className?: string;
  children: React.ReactNode;
}) {
  const { open } = useLightbox();
  return (
    <button type="button" className={className} onClick={() => open(photos, index)}>
      {children}
    </button>
  );
}
