'use client';

import { usePathname } from 'next/navigation';
import { useEffect, useRef, useState, useSyncExternalStore, type CSSProperties } from 'react';
import { zonedToEpoch } from '@/lib/dates';
import { Icon } from './Icon';

/** Adds `.in` to every `[data-reveal]` element as it scrolls into view. */
export function RevealObserver() {
  const pathname = usePathname();
  useEffect(() => {
    const elements = document.querySelectorAll('[data-reveal]:not(.in)');
    if (!('IntersectionObserver' in window)) {
      elements.forEach((el) => el.classList.add('in'));
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.classList.add('in');
            observer.unobserve(entry.target);
          }
        }
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.06 },
    );
    elements.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, [pathname]);
  return null;
}

const pad = (n: number) => String(n).padStart(2, '0');

const subscribeClock = (tick: () => void) => {
  const timer = window.setInterval(tick, 1000);
  return () => window.clearInterval(timer);
};
const clockSnapshot = () => Math.floor(Date.now() / 1000) * 1000;
const serverClock = () => null;

/** Current time, ticking every second; `null` while server rendering / hydrating. */
function useNow(): number | null {
  return useSyncExternalStore(subscribeClock, clockSnapshot, serverClock);
}

/** "在一起的第 N 天" with the seconds of today ticking along. */
export function DaysCounter({ since, initialDay }: { since: string; initialDay: number }) {
  const now = useNow();
  let day = initialDay;
  let clock = '00:00:00';
  if (now !== null) {
    const [y, m, d] = since.split('-').map(Number);
    const start = new Date(y, m - 1, d).getTime();
    const elapsed = Math.max(0, now - start);
    day = Math.floor(elapsed / 86_400_000) + 1;
    const rest = Math.floor((elapsed % 86_400_000) / 1000);
    clock = `${pad(Math.floor(rest / 3600))}:${pad(Math.floor((rest % 3600) / 60))}:${pad(rest % 60)}`;
  }
  return (
    <div className="counter">
      <p className="counter-label">这是我们在一起的第</p>
      <p className="counter-number">
        <span suppressHydrationWarning>{day.toLocaleString('zh-CN')}</span>
        <small>天</small>
      </p>
      <p className="counter-clock" suppressHydrationWarning>
        {now === null ? ' ' : `今天又一起走过了 ${clock}`}
      </p>
    </div>
  );
}

export function WeddingCountdown({ date, time }: { date: string; time: string }) {
  const now = useNow();
  if (now === null) return <p className="countdown">&nbsp;</p>;
  const target = zonedToEpoch(date, time);
  const diff = target - now;
  if (diff <= 0) {
    const days = Math.floor(-diff / 86_400_000);
    return (
      <p className="countdown">
        {days === 0 ? '就是今天！' : `我们已经结婚 ${days.toLocaleString('zh-CN')} 天啦`}
      </p>
    );
  }
  const days = Math.floor(diff / 86_400_000);
  const rest = Math.floor((diff % 86_400_000) / 1000);
  return (
    <p className="countdown">
      距离婚礼还有 <b>{days}</b> 天 <span className="countdown-clock">
        {pad(Math.floor(rest / 3600))}:{pad(Math.floor((rest % 3600) / 60))}:{pad(rest % 60)}
      </span>
    </p>
  );
}

/** Background music; starts on the envelope's `rt:play` event or a tap. */
export function MusicPlayer({ src, autoplayInWeChat }: { src: string; autoplayInWeChat: boolean }) {
  const audio = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    const play = () => {
      el.play().then(
        () => setPlaying(true),
        () => setPlaying(false),
      );
    };
    window.addEventListener('rt:play', play);
    // WeChat allows sound once its JS bridge is ready, the way most H5 invitations do it.
    if (autoplayInWeChat) document.addEventListener('WeixinJSBridgeReady', play, { once: true });
    const onPause = () => setPlaying(false);
    const onPlay = () => setPlaying(true);
    el.addEventListener('pause', onPause);
    el.addEventListener('play', onPlay);
    return () => {
      window.removeEventListener('rt:play', play);
      document.removeEventListener('WeixinJSBridgeReady', play);
      el.removeEventListener('pause', onPause);
      el.removeEventListener('play', onPlay);
    };
  }, [autoplayInWeChat]);

  return (
    <>
      <audio ref={audio} src={src} loop preload="none" />
      <button
        type="button"
        className={`music ${playing ? 'on' : ''}`}
        aria-label={playing ? '暂停音乐' : '播放音乐'}
        onClick={() => {
          const el = audio.current;
          if (!el) return;
          if (el.paused) void el.play().catch(() => {});
          else el.pause();
        }}
      >
        <span className="music-disc">
          <Icon name="music" size={18} />
        </span>
      </button>
    </>
  );
}

type Petal = { id: number; left: number; delay: number; duration: number; size: number; hue: string; sway: number; heart: boolean };
const PETAL_COLORS = ['#f2b8bf', '#e98a95', '#f7d6d0', '#d9505c', '#f3cf9e', '#fbe3e6'];

/** A handful of falling petals, fired by the `rt:petals` event. */
export function Petals() {
  const [petals, setPetals] = useState<Petal[]>([]);
  useEffect(() => {
    let timer = 0;
    const fire = () => {
      if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
      const count = window.innerWidth < 640 ? 22 : 36;
      setPetals(
        Array.from({ length: count }, (_, i) => ({
          id: Date.now() + i,
          left: Math.random() * 100,
          delay: Math.random() * 2.6,
          duration: 5 + Math.random() * 4,
          size: 10 + Math.random() * 14,
          hue: PETAL_COLORS[i % PETAL_COLORS.length],
          sway: 20 + Math.random() * 60,
          heart: i % 4 === 0,
        })),
      );
      window.clearTimeout(timer);
      timer = window.setTimeout(() => setPetals([]), 12_000);
    };
    window.addEventListener('rt:petals', fire);
    return () => {
      window.removeEventListener('rt:petals', fire);
      window.clearTimeout(timer);
    };
  }, []);
  if (petals.length === 0) return null;
  return (
    <div className="petals" aria-hidden>
      {petals.map((p) => (
        <span
          key={p.id}
          className="petal"
          style={
            {
              left: `${p.left}%`,
              '--delay': `${p.delay}s`,
              '--dur': `${p.duration}s`,
              '--size': `${p.size}px`,
              '--sway': `${p.sway}px`,
              color: p.hue,
            } as CSSProperties
          }
        >
          {p.heart ? (
            <Icon name="heart" filled size={p.size} />
          ) : (
            <svg width={p.size} height={p.size} viewBox="0 0 20 20">
              <path d="M10 1c4 4 6 8 0 18C4 9 6 5 10 1z" fill="currentColor" />
            </svg>
          )}
        </span>
      ))}
    </div>
  );
}
