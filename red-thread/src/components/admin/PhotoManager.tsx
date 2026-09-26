'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, useTransition } from 'react';
import { bulkPhotos, type BulkOp } from '@/app/admin/actions';
import { formatTaken } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { FadeImg, blurStyle } from '@/components/Polaroid';
import type { PhotoCard, Visibility } from '@/lib/types';

export type AdminPhoto = PhotoCard & {
  visibility: Visibility;
  featured: boolean;
  favorite: boolean;
  momentTitle: string | null;
};

export function PhotoManager({
  photos,
  moments,
}: {
  photos: AdminPhoto[];
  moments: { id: string; title: string }[];
}) {
  const router = useRouter();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [pending, start] = useTransition();
  const [target, setTarget] = useState('');

  const toggle = (id: string) =>
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const run = (op: BulkOp, momentId?: string) => {
    const ids = [...selected];
    if (op === 'delete' && !window.confirm(`确定删除这 ${ids.length} 张照片吗？存储里的文件也会一起删掉，没法找回。`)) return;
    start(async () => {
      await bulkPhotos(ids, op, momentId ?? null);
      setSelected(new Set());
      router.refresh();
    });
  };

  const allSelected = photos.length > 0 && photos.every((p) => selected.has(p.id));

  return (
    <>
      <div className="pm-tools">
        <label className="check">
          <input
            type="checkbox"
            checked={allSelected}
            onChange={() => setSelected(allSelected ? new Set() : new Set(photos.map((p) => p.id)))}
          />
          全选这一页
        </label>
        <span className="faint small">点照片编辑文字，勾选左上角可以批量整理</span>
      </div>

      <div className="pm-grid">
        {photos.map((photo) => (
          <div key={photo.id} className={`pm-item ${selected.has(photo.id) ? 'selected' : ''}`}>
            <Link href={`/admin/photos/${photo.id}`} className="pm-img" style={blurStyle(photo)}>
              <FadeImg src={photo.thumb} alt={photo.caption ?? ''} />
            </Link>
            <label className="pm-check" aria-label="选择">
              <input type="checkbox" checked={selected.has(photo.id)} onChange={() => toggle(photo.id)} />
            </label>
            <span className="pm-badges">
              {photo.visibility === 'public' ? (
                <span className="pm-badge pub" title="公开">
                  <Icon name="eye" size={12} />
                </span>
              ) : (
                <span className="pm-badge" title="只给彼此">
                  <Icon name="lock" size={12} />
                </span>
              )}
              {photo.featured ? (
                <span className="pm-badge star" title="精选（出现在首页和请柬里）">
                  <Icon name="star" size={12} filled />
                </span>
              ) : null}
              {photo.favorite ? (
                <span className="pm-badge heart" title="我们最爱">
                  <Icon name="heart" size={12} filled />
                </span>
              ) : null}
            </span>
            <span className="pm-meta">
              <span className="pm-caption">{photo.caption || <span className="faint">还没写字</span>}</span>
              <span className="faint">
                {formatTaken(photo.takenAt).slice(0, 11) || '日期未知'}
                {photo.momentTitle ? ` · ${photo.momentTitle}` : ''}
              </span>
            </span>
          </div>
        ))}
      </div>

      {selected.size > 0 ? (
        <div className="pm-bar" role="toolbar" aria-label="批量整理">
          <strong>已选 {selected.size} 张</strong>
          <button type="button" className="btn btn-sm" disabled={pending} onClick={() => run('public')}>
            <Icon name="eye" size={15} /> 公开
          </button>
          <button type="button" className="btn btn-sm" disabled={pending} onClick={() => run('private')}>
            <Icon name="lock" size={15} /> 只给彼此
          </button>
          <button type="button" className="btn btn-sm" disabled={pending} onClick={() => run('feature')}>
            <Icon name="star" size={15} /> 精选
          </button>
          <button type="button" className="btn btn-sm" disabled={pending} onClick={() => run('unfeature')}>
            取消精选
          </button>
          <button type="button" className="btn btn-sm" disabled={pending} onClick={() => run('favorite')}>
            <Icon name="heart" size={15} /> 最爱
          </button>
          <span className="pm-move">
            <select value={target} onChange={(e) => setTarget(e.target.value)} className="input">
              <option value="">放进回忆…</option>
              <option value="__none">移出回忆（不归档）</option>
              {moments.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.title}
                </option>
              ))}
            </select>
            <button
              type="button"
              className="btn btn-sm"
              disabled={pending || !target}
              onClick={() => run('moment', target === '__none' ? undefined : target)}
            >
              移动
            </button>
          </span>
          <Link href={`/admin/moments/new?photos=${[...selected].join(',')}`} className="btn btn-sm">
            <Icon name="plus" size={15} /> 用它们新建回忆
          </Link>
          <button type="button" className="btn btn-sm btn-danger" disabled={pending} onClick={() => run('delete')}>
            <Icon name="trash" size={15} /> 删除
          </button>
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => setSelected(new Set())}>
            取消
          </button>
        </div>
      ) : null}
    </>
  );
}
