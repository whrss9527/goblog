import { MOMENT_KINDS } from '@/lib/moments';
import { saveMomentAction } from '@/app/admin/actions';
import { SaveButton } from '@/components/admin/Small';
import { Icon } from '@/components/Icon';
import type { Moment } from '@/lib/types';

export type CoverOption = { id: string; thumb: string };

export function MomentForm({
  moment,
  defaults,
  covers,
  photoIds,
}: {
  moment: Moment | null;
  defaults?: Partial<Moment>;
  covers: CoverOption[];
  photoIds?: string[];
}) {
  const value = { ...defaults, ...moment };
  return (
    <form action={saveMomentAction.bind(null, moment?.id ?? null)} className="card form-grid">
      {photoIds?.length ? <input type="hidden" name="photoIds" value={photoIds.join(',')} /> : null}
      <label className="field span-2">
        <span>标题</span>
        <input name="title" required maxLength={60} defaultValue={value.title ?? ''} placeholder="比如：第一次一起看海" />
      </label>
      <label className="field">
        <span>类型</span>
        <select name="kind" defaultValue={value.kind ?? 'date'}>
          {MOMENT_KINDS.map((kind) => (
            <option key={kind.value} value={kind.value}>
              {kind.label}
            </option>
          ))}
        </select>
        <small>初遇 / 纪念日 / 里程碑会在红线上系成一颗心。</small>
      </label>
      <label className="field">
        <span>地点</span>
        <input name="place" maxLength={60} defaultValue={value.place ?? ''} placeholder="会变成“去过的地方”里的一枚邮票" />
      </label>
      <label className="field">
        <span>开始日期</span>
        <input type="date" name="startsOn" required defaultValue={value.startsOn ?? ''} />
      </label>
      <label className="field">
        <span>结束日期（可选）</span>
        <input type="date" name="endsOn" defaultValue={value.endsOn ?? ''} />
      </label>
      <label className="field span-2">
        <span>那天的故事</span>
        <textarea name="story" rows={6} maxLength={3000} defaultValue={value.story ?? ''} placeholder="写给以后的我们，也写给来看的朋友。" />
      </label>
      <fieldset className="field span-2 radio-row">
        <legend className="field-label">谁能看到</legend>
        <label className="check">
          <input type="radio" name="visibility" value="private" defaultChecked={value.visibility !== 'public'} />
          <Icon name="lock" size={15} /> 只给彼此
        </label>
        <label className="check">
          <input type="radio" name="visibility" value="public" defaultChecked={value.visibility === 'public'} />
          <Icon name="eye" size={15} /> 写进公开的故事里
        </label>
        <small>公开的回忆里，只有设为“公开”的照片会给客人看；其余的会显示成“还有 N 张，只给彼此看”。</small>
      </fieldset>
      {covers.length > 0 ? (
        <fieldset className="field span-2">
          <legend className="field-label">封面（放在最上面的那张）</legend>
          <div className="cover-pick">
            <label>
              <input type="radio" name="coverPhotoId" value="" defaultChecked={!value.coverPhotoId} />
              <span className="cover-auto">自动</span>
            </label>
            {covers.map((cover) => (
              <label key={cover.id}>
                <input type="radio" name="coverPhotoId" value={cover.id} defaultChecked={value.coverPhotoId === cover.id} />
                <img src={cover.thumb} alt="" loading="lazy" />
              </label>
            ))}
          </div>
        </fieldset>
      ) : null}
      <div className="span-2 form-actions">
        <SaveButton>{moment ? '保存' : '写进相册'}</SaveButton>
      </div>
    </form>
  );
}
