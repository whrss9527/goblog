'use client';

import { useCallback, useEffect, useState, useSyncExternalStore } from 'react';

const OPENED_KEY = 'rt-envelope-opened';

const noSubscribe = () => () => {};
function readOpened() {
  try {
    return sessionStorage.getItem(OPENED_KEY) === '1';
  } catch {
    return false;
  }
}

/**
 * The first thing a guest sees: a sealed envelope addressed to them.
 * Breaking the wax seal opens the flap, the letter slides out, petals fall
 * and — because this tap is a user gesture — the music may start.
 */
export function Envelope({
  to,
  line,
  initials,
  names,
  letterTop,
  letterBottom,
}: {
  to: string | null;
  line: string;
  initials: string;
  names: string;
  letterTop: string;
  letterBottom: string;
}) {
  const [stage, setStage] = useState<'sealed' | 'opening' | 'letter' | 'leaving' | 'gone'>('sealed');

  // Guests who already opened it this visit go straight to the album,
  // unless they arrived through a personal link addressed to them.
  const opened = useSyncExternalStore(noSubscribe, readOpened, () => false);
  const hidden = stage === 'gone' || (stage === 'sealed' && opened && !to);

  useEffect(() => {
    if (hidden || stage === 'leaving') return;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = overflow;
    };
  }, [hidden, stage]);

  const open = useCallback(() => {
    if (stage !== 'sealed') return;
    try {
      sessionStorage.setItem(OPENED_KEY, '1');
    } catch {}
    window.dispatchEvent(new Event('rt:play'));
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reduced) {
      setStage('gone');
      return;
    }
    setStage('opening');
    window.setTimeout(() => setStage('letter'), 900);
    window.setTimeout(() => {
      setStage('leaving');
      window.dispatchEvent(new Event('rt:petals'));
    }, 3300);
    window.setTimeout(() => setStage('gone'), 4200);
  }, [stage]);

  const skip = () => {
    try {
      sessionStorage.setItem(OPENED_KEY, '1');
    } catch {}
    setStage('gone');
  };

  if (hidden) return null;

  return (
    <div className={`env-overlay stage-${stage}`} role="dialog" aria-modal="true" aria-label="一封信">
      <div className="env-wrap">
        <p className="env-to">
          {to ? (
            <>
              <span className="env-to-label">致</span> {to}
            </>
          ) : (
            line
          )}
        </p>

        <button type="button" className="env" onClick={open} aria-label="拆开信封" disabled={stage !== 'sealed'}>
          <span className="env-back" />
          <span className="env-letter">
            <span className="env-letter-top script">{letterTop}</span>
            <span className="env-letter-names">{names}</span>
            <span className="env-letter-bottom">{letterBottom}</span>
          </span>
          <span className="env-front" />
          <span className="env-flap" />
          <span className="env-seal">
            <span>{initials}</span>
          </span>
        </button>

        <p className="env-hint">{stage === 'sealed' ? '轻触火漆，拆开这封信' : ' '}</p>
      </div>
      {stage === 'sealed' ? (
        <button type="button" className="env-skip" onClick={skip}>
          直接看相册
        </button>
      ) : null}
    </div>
  );
}
