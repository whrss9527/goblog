'use client';

import { useCallback, type CSSProperties } from 'react';
import type { PhotoCard } from '@/lib/types';
import { tapeFor, tiltFor } from '@/lib/ui';
import { Icon } from './Icon';

/** An <img> that fades in once loaded, over the photo's tiny blurred preview. */
export function FadeImg({
  src,
  alt,
  width,
  height,
  eager = false,
  className,
}: {
  src: string;
  alt: string;
  width?: number;
  height?: number;
  eager?: boolean;
  className?: string;
}) {
  // Cached images can finish before hydration, when onLoad has no listener yet.
  const ref = useCallback((img: HTMLImageElement | null) => {
    if (img?.complete && img.naturalWidth > 0) img.classList.add('loaded');
  }, []);
  return (
    <img
      ref={ref}
      src={src}
      alt={alt}
      width={width}
      height={height}
      className={className}
      loading={eager ? 'eager' : 'lazy'}
      decoding="async"
      draggable={false}
      onLoad={(event) => event.currentTarget.classList.add('loaded')}
      onError={(event) => event.currentTarget.classList.add('loaded')}
    />
  );
}

export const blurStyle = (photo: Pick<PhotoCard, 'blurData' | 'color'>): CSSProperties => ({
  backgroundImage: photo.blurData ? `url("${photo.blurData}")` : undefined,
  backgroundColor: photo.color ?? undefined,
});

export function Polaroid({
  photo,
  onOpen,
  tape = true,
  tilt,
  eager,
  showLock,
  className,
  style,
  caption,
}: {
  photo: PhotoCard;
  onOpen?: () => void;
  tape?: boolean;
  tilt?: string;
  eager?: boolean;
  showLock?: boolean;
  className?: string;
  style?: CSSProperties;
  caption?: string | null;
}) {
  const body = (
    <>
      {tape ? (
        <span
          className="tape"
          style={{ '--tape': tapeFor(photo.id), '--tape-tilt': tiltFor(photo.id + 't', 6) } as CSSProperties}
        />
      ) : null}
      <span className="polaroid-img" style={blurStyle(photo)}>
        <FadeImg src={photo.thumb} alt={photo.caption ?? ''} eager={eager} />
      </span>
      <span className="polaroid-cap">{caption === undefined ? photo.caption : caption}</span>
      {showLock && photo.isPrivate ? (
        <span className="lock-badge" title="只有我们看得到">
          <Icon name="lock" size={14} />
        </span>
      ) : null}
    </>
  );
  const merged = { '--tilt': tilt ?? tiltFor(photo.id), ...style } as CSSProperties;
  if (onOpen) {
    return (
      <button type="button" className={`polaroid ${className ?? ''}`} style={merged} onClick={onOpen}>
        {body}
      </button>
    );
  }
  return (
    <span className={`polaroid ${className ?? ''}`} style={merged}>
      {body}
    </span>
  );
}
