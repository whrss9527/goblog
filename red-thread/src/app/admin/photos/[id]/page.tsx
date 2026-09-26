import Link from 'next/link';
import { notFound } from 'next/navigation';
import { requireSession } from '@/lib/auth';
import { listMoments } from '@/lib/moments';
import { getPhoto } from '@/lib/photos';
import { getSettings } from '@/lib/settings';
import { urlFor } from '@/lib/storage';
import { deletePhotoAction, updatePhotoAction } from '@/app/admin/actions';
import { ConfirmSubmit, SaveButton } from '@/components/admin/Small';
import { Icon } from '@/components/Icon';

export const dynamic = 'force-dynamic';

export default async function EditPhoto({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  await requireSession(`/admin/photos/${id}`);
  const photo = await getPhoto(id);
  if (!photo) notFound();
  const [settings, moments] = await Promise.all([getSettings(), listMoments({ newestFirst: true })]);
  const isPrivate = photo.visibility === 'private';
  const [src, original] = await Promise.all([
    urlFor(photo.lgKey, isPrivate),
    photo.origKey ? urlFor(photo.origKey, true) : Promise.resolve(null),
  ]);
  const hasGps = photo.latitude !== null && photo.longitude !== null;

  return (
    <div>
      <Link href="/admin/photos" className="muted back-link">
        ← 所有照片
      </Link>
      <div className="edit-photo">
        <figure className="edit-preview">
          <span className="polaroid" style={{ '--tilt': '-1.5deg' } as React.CSSProperties}>
            <span className="edit-img" style={{ aspectRatio: `${photo.width} / ${photo.height}`, backgroundColor: photo.color ?? undefined }}>
              <img src={src} alt={photo.caption ?? ''} />
            </span>
            <span className="polaroid-cap">{photo.caption}</span>
          </span>
          <figcaption className="edit-facts">
            <span>
              {photo.width} × {photo.height}
            </span>
            {photo.camera ? <span>{photo.camera}</span> : null}
            {hasGps ? (
              <a
                href={`https://uri.amap.com/marker?position=${photo.longitude},${photo.latitude}&coordinate=wgs84&name=${encodeURIComponent(photo.caption ?? '拍照的地方')}`}
                target="_blank"
                rel="noreferrer"
              >
                <Icon name="pin" size={14} /> 在地图上看拍摄地点
              </a>
            ) : null}
            {original ? (
              <a href={original} target="_blank" rel="noreferrer">
                <Icon name="download" size={14} /> 下载原图
              </a>
            ) : null}
          </figcaption>
        </figure>

        <form action={updatePhotoAction.bind(null, photo.id)} className="card form-grid">
          <input type="hidden" name="back" value="/admin/photos" />
          <label className="field span-2">
            <span>写在正面的话</span>
            <input name="caption" defaultValue={photo.caption ?? ''} maxLength={60} placeholder="一句话，会用手写体写在拍立得下方" />
          </label>
          <label className="field span-2">
            <span>写在背面的话</span>
            <textarea name="note" defaultValue={photo.note ?? ''} maxLength={600} rows={4} placeholder="翻到照片背面才能看到的悄悄话。公开的照片，客人也能翻过来看。" />
          </label>
          <label className="field">
            <span>拍摄时间</span>
            <input type="datetime-local" name="takenAt" defaultValue={photo.takenAt?.slice(0, 16) ?? ''} />
          </label>
          <label className="field">
            <span>地点</span>
            <input name="place" defaultValue={photo.place ?? ''} maxLength={60} placeholder="比如：杭州 · 西湖" />
          </label>
          <label className="field">
            <span>属于哪段回忆</span>
            <select name="momentId" defaultValue={photo.momentId ?? ''}>
              <option value="">不归档</option>
              {moments.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.title}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span>谁拍的</span>
            <select name="author" defaultValue={photo.author ?? ''}>
              <option value="">不写</option>
              <option value="a">{settings.partnerA}</option>
              <option value="b">{settings.partnerB}</option>
            </select>
          </label>
          <fieldset className="field span-2 radio-row">
            <legend className="field-label">谁能看到</legend>
            <label className="check">
              <input type="radio" name="visibility" value="private" defaultChecked={isPrivate} />
              <Icon name="lock" size={15} /> 只给彼此
            </label>
            <label className="check">
              <input type="radio" name="visibility" value="public" defaultChecked={!isPrivate} />
              <Icon name="eye" size={15} /> 公开给客人
            </label>
          </fieldset>
          <div className="span-2 radio-row">
            <label className="check">
              <input type="checkbox" name="featured" defaultChecked={photo.featured} />
              <Icon name="star" size={15} /> 精选：放在首页、信封和请柬里（需要公开）
            </label>
            <label className="check">
              <input type="checkbox" name="favorite" defaultChecked={photo.favorite} />
              <Icon name="heart" size={15} /> 我们最爱
            </label>
          </div>
          <div className="span-2 form-actions">
            <SaveButton />
          </div>
        </form>
      </div>

      <form action={deletePhotoAction.bind(null, photo.id)} className="danger-zone">
        <ConfirmSubmit message="确定删除这张照片吗？存储里的文件也会一起删掉，没法找回。">删除这张照片</ConfirmSubmit>
      </form>
    </div>
  );
}
