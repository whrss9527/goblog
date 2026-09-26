'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useRef, useState } from 'react';
import { createPhoto } from '@/app/admin/actions';
import { newId, newToken } from '@/lib/ids';
import { preparePhoto, uploadFiles } from '@/lib/image-client';
import type { Visibility } from '@/lib/types';
import { Icon } from '@/components/Icon';

type Item = {
  uid: string;
  file: File;
  preview: string | null;
  status: 'waiting' | 'preparing' | 'uploading' | 'saving' | 'done' | 'error';
  progress: number;
  error?: string;
};

const ACCEPT = 'image/jpeg,image/png,image/webp,image/heic,image/heif,image/avif,.heic,.heif';
const CONCURRENCY = 2;

const extOf = (file: File) => {
  const ext = file.name.split('.').pop()?.toLowerCase() ?? 'jpg';
  return ['jpg', 'jpeg', 'png', 'webp', 'heic', 'heif', 'gif', 'avif'].includes(ext) ? ext : 'jpg';
};
const typeOf = (file: File, ext: string) =>
  file.type || (ext === 'heic' || ext === 'heif' ? `image/${ext}` : ext === 'jpg' ? 'image/jpeg' : `image/${ext}`);

const STATUS: Record<Item['status'], string> = {
  waiting: '排队中',
  preparing: '读取照片…',
  uploading: '上传中',
  saving: '收进相册…',
  done: '放好啦',
  error: '出错了',
};

export function Uploader({
  moments,
  defaultMomentId,
}: {
  moments: { id: string; title: string }[];
  defaultMomentId: string | null;
}) {
  const router = useRouter();
  const [items, setItems] = useState<Item[]>([]);
  const [momentId, setMomentId] = useState(defaultMomentId ?? '');
  const [visibility, setVisibility] = useState<Visibility>('private');
  const [place, setPlace] = useState('');
  const [keepOriginal, setKeepOriginal] = useState(false);
  const [dragging, setDragging] = useState(false);
  const running = useRef(0);
  const options = useRef({ momentId, visibility, place, keepOriginal });
  useEffect(() => {
    options.current = { momentId, visibility, place, keepOriginal };
  }, [momentId, visibility, place, keepOriginal]);
  const input = useRef<HTMLInputElement>(null);

  const patch = (uid: string, change: Partial<Item>) =>
    setItems((list) => list.map((item) => (item.uid === uid ? { ...item, ...change } : item)));

  const process = useCallback(async (item: Item) => {
    const opts = options.current;
    try {
      patch(item.uid, { status: 'preparing' });
      const prepared = await preparePhoto(item.file);
      const id = newId();
      const token = newToken();
      const base = `photos/${id}-${token}`;
      const ext = extOf(item.file);
      const files = [
        { key: `${base}-lg.jpg`, blob: prepared.lg },
        { key: `${base}-sm.jpg`, blob: prepared.sm },
      ];
      if (opts.keepOriginal) {
        files.push({
          key: `${base}-orig.${ext}`,
          blob: item.file.type ? item.file : new Blob([item.file], { type: typeOf(item.file, ext) }),
        });
      }
      patch(item.uid, { status: 'uploading', progress: 0, preview: URL.createObjectURL(prepared.sm) });
      await uploadFiles(files, (progress) => patch(item.uid, { progress }));
      patch(item.uid, { status: 'saving', progress: 1 });
      const result = await createPhoto({
        id,
        lgKey: files[0].key,
        smKey: files[1].key,
        origKey: files[2]?.key ?? null,
        width: prepared.width,
        height: prepared.height,
        blurData: prepared.blurData,
        color: prepared.color,
        takenAt: prepared.takenAt,
        latitude: prepared.latitude,
        longitude: prepared.longitude,
        camera: prepared.camera,
        caption: null,
        place: opts.place || null,
        momentId: opts.momentId || null,
        visibility: opts.visibility,
      });
      if (!result.ok) throw new Error(result.error);
      patch(item.uid, { status: 'done' });
    } catch (error) {
      patch(item.uid, { status: 'error', error: (error as Error).message || '未知错误' });
    }
  }, []);

  // A tiny queue: at most CONCURRENCY photos are being worked on at once.
  useEffect(() => {
    const free = CONCURRENCY - running.current;
    const next = items.filter((item) => item.status === 'waiting').slice(0, Math.max(0, free));
    for (const item of next) {
      running.current += 1;
      patch(item.uid, { status: 'preparing' });
      void process(item).finally(() => {
        running.current -= 1;
        setItems((list) => [...list]);
      });
    }
  }, [items, process]);

  const add = (files: FileList | File[] | null) => {
    if (!files) return;
    const list = Array.from(files).filter((file) => file.type.startsWith('image/') || /\.(heic|heif)$/i.test(file.name));
    setItems((current) => [
      ...current,
      ...list.map((file) => ({
        uid: `${file.name}-${file.size}-${Math.random().toString(36).slice(2)}`,
        file,
        preview: null,
        status: 'waiting' as const,
        progress: 0,
      })),
    ]);
  };

  const done = items.filter((item) => item.status === 'done').length;
  const failed = items.filter((item) => item.status === 'error').length;
  const busy = items.some((item) => !['done', 'error'].includes(item.status));

  useEffect(() => {
    if (!busy) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [busy]);

  return (
    <div className="uploader">
      <div className="upload-options card">
        <label className="field">
          <span>放进哪段回忆</span>
          <select value={momentId} onChange={(e) => setMomentId(e.target.value)}>
            <option value="">暂不归档（之后再整理）</option>
            {moments.map((m) => (
              <option key={m.id} value={m.id}>
                {m.title}
              </option>
            ))}
          </select>
        </label>
        <div className="field">
          <span>谁能看到</span>
          <div className="seg">
            <button type="button" className={visibility === 'private' ? 'on' : ''} onClick={() => setVisibility('private')}>
              <Icon name="lock" size={15} /> 只给彼此
            </button>
            <button type="button" className={visibility === 'public' ? 'on' : ''} onClick={() => setVisibility('public')}>
              <Icon name="eye" size={15} /> 公开给客人
            </button>
          </div>
        </div>
        <label className="field">
          <span>地点（可选）</span>
          <input value={place} onChange={(e) => setPlace(e.target.value)} placeholder="比如：厦门 · 鼓浪屿" maxLength={60} />
        </label>
        <label className="check">
          <input type="checkbox" checked={keepOriginal} onChange={(e) => setKeepOriginal(e.target.checked)} />
          同时保存原图（占用更多存储，原图只有我们能下载）
        </label>
      </div>

      <div
        className={`dropzone ${dragging ? 'over' : ''}`}
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          add(e.dataTransfer.files);
        }}
        onClick={() => input.current?.click()}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') input.current?.click();
        }}
      >
        <Icon name="upload" size={34} />
        <p className="hand dropzone-title">把回忆拖到这里</p>
        <p className="muted">或者点一下选择照片 · 支持 JPG / PNG / HEIC · 可以一次选很多张</p>
        <input
          ref={input}
          type="file"
          accept={ACCEPT}
          multiple
          hidden
          onChange={(e) => {
            add(e.target.files);
            e.target.value = '';
          }}
        />
      </div>

      {items.length > 0 ? (
        <>
          <div className="upload-summary">
            <span>
              {done} / {items.length} 张已放好{failed ? `，${failed} 张失败` : ''}
            </span>
            {!busy && done > 0 ? (
              <span className="upload-next">
                <Link href={momentId ? `/us/moments/${momentId}` : '/admin/photos?filter=recent'} className="btn btn-red btn-sm">
                  去看看
                </Link>
                <button type="button" className="btn btn-sm" onClick={() => router.push('/admin/photos')}>
                  整理照片
                </button>
              </span>
            ) : null}
          </div>
          <ul className="upload-list">
            {items.map((item) => (
              <li key={item.uid} className={`upload-item ${item.status}`}>
                <span className="upload-thumb">
                  {item.preview ? <img src={item.preview} alt="" /> : <Icon name="image" size={22} />}
                </span>
                <span className="upload-name">{item.file.name}</span>
                <span className="upload-status">
                  {item.status === 'uploading' ? `${Math.round(item.progress * 100)}%` : STATUS[item.status]}
                  {item.error ? ` · ${item.error}` : ''}
                </span>
                <span className="upload-bar" style={{ transform: `scaleX(${item.status === 'done' ? 1 : item.progress})` }} />
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </div>
  );
}
