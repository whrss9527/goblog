'use client';

import { useEffect, useMemo, useState } from 'react';
import QRCode from 'qrcode';
import { Icon } from '@/components/Icon';

function useQr(text: string, width = 480) {
  const [data, setData] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    QRCode.toDataURL(text, {
      width,
      margin: 1,
      errorCorrectionLevel: 'M',
      color: { dark: '#3a2e2aff', light: '#ffffffff' },
    }).then((url) => alive && setData(url), () => alive && setData(null));
    return () => {
      alive = false;
    };
  }, [text, width]);
  return data;
}

function CopyButton({ text, label = '复制' }: { text: string; label?: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className="btn btn-sm"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
        } catch {
          const area = document.createElement('textarea');
          area.value = text;
          document.body.appendChild(area);
          area.select();
          document.execCommand('copy');
          area.remove();
        }
        setDone(true);
        window.setTimeout(() => setDone(false), 1600);
      }}
    >
      <Icon name={done ? 'check' : 'link'} size={15} />
      {done ? '已复制' : label}
    </button>
  );
}

function Qr({ text, name, size = 132 }: { text: string; name: string; size?: number }) {
  const data = useQr(text);
  if (!data) return <span className="qr-box" style={{ width: size, height: size }} />;
  return (
    <a href={data} download={`${name}.png`} className="qr-box" title="点击下载二维码图片">
      <img src={data} alt={`${name} 的二维码`} width={size} height={size} />
    </a>
  );
}

export function SharePanel({ base, hasWedding }: { base: string; hasWedding: boolean }) {
  const [names, setNames] = useState('');
  const [transparent, setTransparent] = useState(false);
  const guests = useMemo(
    () =>
      [...new Set(names.split(/[\n,，、]/).map((n) => n.trim()).filter(Boolean))]
        .slice(0, 300)
        .map((name) => ({ name: name.slice(0, 20), url: `${base}/?to=${encodeURIComponent(name.slice(0, 20))}` })),
    [names, base],
  );
  const embedUrl = `${base}/embed${transparent ? '?bg=transparent' : ''}`;
  const snippet = `<iframe src="${embedUrl}" width="360" height="600" style="border:0;max-width:100%" loading="lazy" title="我们的相册"></iframe>`;

  return (
    <div className="share">
      <section className="card share-block">
        <h2>公开相册</h2>
        <div className="share-main">
          <Qr text={base + '/'} name="相册二维码" size={160} />
          <div className="share-copy">
            <code className="share-url">{base}/</code>
            <div className="row">
              <CopyButton text={base + '/'} label="复制链接" />
              <a className="btn btn-sm" href="/" target="_blank" rel="noreferrer">
                <Icon name="eye" size={15} /> 打开看看
              </a>
            </div>
            <p className="small muted">二维码可以直接印在纸质请柬上。点二维码下载图片。</p>
          </div>
        </div>
      </section>

      <section className="card share-block">
        <h2>给每位客人一封专属的信</h2>
        <p className="small muted">
          每行一个名字（也可以用逗号隔开）。客人打开自己的链接，信封上会写着“致 某某”
          {hasWedding ? '，请柬上也会写“诚挚邀请 某某”' : ''}。
        </p>
        <textarea
          className="input"
          rows={4}
          value={names}
          onChange={(e) => setNames(e.target.value)}
          placeholder={'王小明\n李阿姨一家\n大学室友们'}
        />
        {guests.length > 0 ? (
          <>
            <div className="row share-bulk">
              <CopyButton text={guests.map((g) => `${g.name}：${g.url}`).join('\n')} label={`复制全部 ${guests.length} 条`} />
            </div>
            <ul className="guest-links">
              {guests.map((guest) => (
                <li key={guest.name}>
                  <Qr text={guest.url} name={`致${guest.name}`} size={84} />
                  <span className="guest-link-main">
                    <strong>致 {guest.name}</strong>
                    <code>{`${base}/?to=${guest.name}`}</code>
                  </span>
                  <CopyButton text={guest.url} />
                </li>
              ))}
            </ul>
          </>
        ) : null}
      </section>

      <section className="card share-block">
        <h2>放进电子请柬</h2>
        <p className="small muted">
          支持嵌入网页的请柬（或者你自己的博客）可以用这段代码放一叠会自己翻动的拍立得；不支持嵌入的，就放上面的链接或二维码。
        </p>
        <label className="check">
          <input type="checkbox" checked={transparent} onChange={(e) => setTransparent(e.target.checked)} /> 透明背景
        </label>
        <div className="embed-row">
          <div className="embed-code">
            <code>{snippet}</code>
            <CopyButton text={snippet} label="复制代码" />
          </div>
          <iframe src={transparent ? '/embed?bg=transparent' : '/embed'} title="嵌入预览" className="embed-preview" />
        </div>
      </section>

      {hasWedding ? (
        <section className="card share-block">
          <h2>日历提醒</h2>
          <p className="small muted">请柬里已经有“加入日历”按钮，也可以单独把这个链接发给长辈：</p>
          <div className="row">
            <code className="share-url">{base}/wedding.ics</code>
            <CopyButton text={`${base}/wedding.ics`} />
          </div>
        </section>
      ) : null}
    </div>
  );
}
