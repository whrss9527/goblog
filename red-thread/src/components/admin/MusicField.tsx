'use client';

import { useState } from 'react';
import { newToken } from '@/lib/ids';
import { uploadFiles } from '@/lib/image-client';
import { Icon } from '@/components/Icon';

/** Background music: paste a link, or upload an audio file to our storage. */
export function MusicField({ initial, previewUrl }: { initial: string; previewUrl: string | null }) {
  const [value, setValue] = useState(initial);
  const [status, setStatus] = useState<string | null>(null);
  const [preview, setPreview] = useState(previewUrl);

  const upload = async (file: File) => {
    const ext = (file.name.split('.').pop() ?? '').toLowerCase();
    if (!['mp3', 'm4a', 'aac', 'ogg', 'wav'].includes(ext)) {
      setStatus('支持 mp3 / m4a / aac / ogg / wav');
      return;
    }
    const key = `media/${newToken()}.${ext}`;
    const type = file.type || (ext === 'mp3' ? 'audio/mpeg' : `audio/${ext}`);
    try {
      setStatus('上传中 0%');
      await uploadFiles([{ key, blob: new Blob([file], { type }) }], (p) => setStatus(`上传中 ${Math.round(p * 100)}%`));
      setValue(key);
      setPreview(URL.createObjectURL(file));
      setStatus('上传好了，记得点下面的“保存”');
    } catch (error) {
      setStatus((error as Error).message);
    }
  };

  return (
    <div className="field span-2">
      <span>背景音乐</span>
      <div className="music-field">
        <input name="music" value={value} onChange={(e) => setValue(e.target.value)} placeholder="https://… 的音频链接，或者上传一首" />
        <label className="btn btn-sm">
          <Icon name="music" size={15} /> 上传
          <input type="file" accept="audio/*" hidden onChange={(e) => e.target.files?.[0] && upload(e.target.files[0])} />
        </label>
        {value ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => { setValue(''); setPreview(null); }}>
            不要音乐
          </button>
        ) : null}
      </div>
      {preview ? <audio src={preview} controls preload="none" className="music-preview" /> : null}
      <small>{status ?? '客人拆开信封时开始播放，右下角可以随时暂停。选一首你们的歌吧。'}</small>
    </div>
  );
}
