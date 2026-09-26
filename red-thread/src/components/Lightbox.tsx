'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from 'react';
import type { PhotoCard } from '@/lib/types';
import { formatTaken } from '@/lib/dates';
import { Icon } from './Icon';
import { blurStyle } from './Polaroid';

type LightboxApi = { open: (photos: PhotoCard[], index?: number) => void };

const LightboxContext = createContext<LightboxApi>({ open: () => {} });

export const useLightbox = () => useContext(LightboxContext);

export function LightboxProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ photos: PhotoCard[]; index: number } | null>(null);
  const api = useMemo<LightboxApi>(
    () => ({
      open: (photos, index = 0) => {
        if (photos.length > 0) setState({ photos, index: Math.min(Math.max(index, 0), photos.length - 1) });
      },
    }),
    [],
  );
  return (
    <LightboxContext.Provider value={api}>
      {children}
      {state ? (
        <Lightbox
          photos={state.photos}
          index={state.index}
          onIndex={(index) => setState((s) => (s ? { ...s, index } : s))}
          onClose={() => setState(null)}
        />
      ) : null}
    </LightboxContext.Provider>
  );
}

type Burst = { id: number; x: number; y: number };

function Lightbox({
  photos,
  index,
  onIndex,
  onClose,
}: {
  photos: PhotoCard[];
  index: number;
  onIndex: (index: number) => void;
  onClose: () => void;
}) {
  const photo = photos[index];
  const [flipped, setFlipped] = useState(false);
  const [bursts, setBursts] = useState<Burst[]>([]);
  const closeRef = useRef<HTMLButtonElement>(null);
  const swipe = useRef<{ x: number; y: number } | null>(null);
  const many = photos.length > 1;

  const go = useCallback(
    (delta: number) => {
      if (!many) return;
      setFlipped(false);
      onIndex((index + delta + photos.length) % photos.length);
    },
    [index, many, onIndex, photos.length],
  );

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    closeRef.current?.focus();
    return () => {
      document.body.style.overflow = overflow;
      previous?.focus?.();
    };
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
      else if (event.key === 'ArrowRight') go(1);
      else if (event.key === 'ArrowLeft') go(-1);
      else if ((event.key === ' ' || event.key === 'f') && photo.note) {
        event.preventDefault();
        setFlipped((f) => !f);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [go, onClose, photo.note]);

  // Warm the neighbours so swiping feels instant.
  useEffect(() => {
    for (const offset of [1, -1]) {
      const next = photos[(index + offset + photos.length) % photos.length];
      if (next) new Image().src = next.src;
    }
  }, [index, photos]);

  const heart = (x: number, y: number) => {
    const id = Date.now() + Math.random();
    setBursts((list) => [...list, { id, x, y }]);
    window.setTimeout(() => setBursts((list) => list.filter((b) => b.id !== id)), 1100);
  };

  const ratio = photo.width / photo.height;
  const meta = [formatTaken(photo.takenAt), photo.place, photo.author ? `by ${photo.author}` : null].filter(Boolean);

  return (
    <div
      className="lb"
      role="dialog"
      aria-modal="true"
      aria-label={photo.caption || '照片'}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
      onPointerDown={(event) => {
        swipe.current = { x: event.clientX, y: event.clientY };
      }}
      onPointerUp={(event) => {
        const start = swipe.current;
        swipe.current = null;
        if (!start) return;
        const dx = event.clientX - start.x;
        const dy = event.clientY - start.y;
        if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy) * 1.5) go(dx < 0 ? 1 : -1);
        else if (dy > 110 && Math.abs(dy) > Math.abs(dx) * 1.5) onClose();
      }}
    >
      <div className="lb-top">
        <span className="lb-count">{many ? `${index + 1} / ${photos.length}` : ''}</span>
        <button ref={closeRef} type="button" className="icon-btn" onClick={onClose} aria-label="关闭">
          <Icon name="close" />
        </button>
      </div>

      <div className="lb-stage" style={{ '--ratio': ratio } as CSSProperties}>
        <div className={`lb-card ${flipped ? 'flipped' : ''}`} key={photo.id}>
          <div className="lb-face lb-front">
            <div
              className="lb-photo"
              style={blurStyle(photo)}
              onDoubleClick={(event) => {
                const rect = event.currentTarget.getBoundingClientRect();
                heart(event.clientX - rect.left, event.clientY - rect.top);
              }}
            >
              <img src={photo.src} alt={photo.caption ?? ''} draggable={false} />
              {bursts.map((burst) => (
                <span key={burst.id} className="burst" style={{ left: burst.x, top: burst.y }} aria-hidden>
                  <Icon name="heart" filled size={56} className="burst-main" />
                  {Array.from({ length: 8 }, (_, i) => (
                    <span key={i} className="burst-bit" style={{ '--a': `${i * 45 + 20}deg` } as CSSProperties}>
                      <Icon name="heart" filled size={14} />
                    </span>
                  ))}
                </span>
              ))}
            </div>
            <p className="lb-caption">{photo.caption}</p>
          </div>
          <div className="lb-face lb-back" aria-hidden={!flipped}>
            <p className="lb-back-label">写在背面的话</p>
            <p className="lb-note">{photo.note}</p>
            <p className="lb-back-foot">
              {photo.takenAt ? formatTaken(photo.takenAt).slice(0, 11) : ''}
              {photo.author ? ` · ${photo.author}` : ''}
            </p>
          </div>
        </div>
      </div>

      <div className="lb-bottom">
        <p className="lb-meta">{meta.join(' · ')}</p>
        <div className="lb-actions">
          {many ? (
            <button type="button" className="icon-btn" onClick={() => go(-1)} aria-label="上一张">
              <Icon name="left" />
            </button>
          ) : null}
          {photo.note ? (
            <button type="button" className="btn btn-sm lb-flip" onClick={() => setFlipped((f) => !f)}>
              <Icon name="flip" size={16} />
              {flipped ? '翻回正面' : '翻到背面'}
            </button>
          ) : (
            <span className="lb-tip">双击照片，送一颗心</span>
          )}
          {many ? (
            <button type="button" className="icon-btn" onClick={() => go(1)} aria-label="下一张">
              <Icon name="right" />
            </button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
