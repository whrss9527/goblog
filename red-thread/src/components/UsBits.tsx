'use client';

import { useState, useTransition } from 'react';
import { drawMemory } from '@/app/us/actions';
import type { PhotoCard } from '@/lib/types';
import { useLightbox } from './Lightbox';
import { FadeImg, blurStyle } from './Polaroid';
import { Icon } from './Icon';

export function DrawMemoryButton() {
  const { open } = useLightbox();
  const [pending, start] = useTransition();
  const [empty, setEmpty] = useState(false);
  return (
    <button
      type="button"
      className="btn btn-red"
      disabled={pending}
      onClick={() =>
        start(async () => {
          const card = await drawMemory();
          if (card) open([card]);
          else setEmpty(true);
        })
      }
    >
      <Icon name="shuffle" size={18} />
      {pending ? '翻找中…' : empty ? '还没有照片呢' : '抽一张回忆'}
    </button>
  );
}

export type CalendarCell = {
  day: string | null;
  photos: PhotoCard[];
  marks: string[];
  isToday: boolean;
};

/** Month grid where every day we took photos shows one of them. */
export function PhotoCalendar({ cells }: { cells: CalendarCell[] }) {
  const { open } = useLightbox();
  return (
    <div className="cal">
      {['一', '二', '三', '四', '五', '六', '日'].map((name) => (
        <div key={name} className="cal-head">
          {name}
        </div>
      ))}
      {cells.map((cell, i) => {
        if (!cell.day) return <div key={i} className="cal-cell cal-pad" />;
        const cover = cell.photos[0];
        const classes = ['cal-cell', cover ? 'has-photo' : '', cell.marks.length ? 'marked' : '', cell.isToday ? 'today' : '']
          .filter(Boolean)
          .join(' ');
        const inner = (
          <>
            {cover ? (
              <span className="cal-img" style={blurStyle(cover)}>
                <FadeImg src={cover.thumb} alt="" />
              </span>
            ) : null}
            <span className="cal-num">{Number(cell.day.slice(8))}</span>
            {cell.photos.length > 1 ? <span className="cal-count">{cell.photos.length}</span> : null}
            {cell.marks.length ? (
              <span className="cal-mark">
                <Icon name="heart" size={10} filled />
                {cell.marks[0]}
              </span>
            ) : null}
          </>
        );
        return cover ? (
          <button
            key={cell.day}
            type="button"
            className={classes}
            onClick={() => open(cell.photos)}
            aria-label={`${cell.day}，${cell.photos.length} 张照片`}
          >
            {inner}
          </button>
        ) : (
          <div key={cell.day} className={classes} title={cell.marks.join('、') || undefined}>
            {inner}
          </div>
        );
      })}
    </div>
  );
}
