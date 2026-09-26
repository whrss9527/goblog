'use client';

import { useActionState, useState } from 'react';
import { submitGuestNote, type GuestFormState } from '@/app/guest-actions';
import { Icon } from './Icon';

export function GuestbookForm({ rsvp, deadline }: { rsvp: boolean; deadline: string | null }) {
  const [state, action, pending] = useActionState<GuestFormState, FormData>(submitGuestNote, { status: 'idle' });
  const [attending, setAttending] = useState<string>('');

  if (state.status === 'ok') {
    return (
      <div className="gb-thanks" role="status">
        <span className="gb-plane" aria-hidden>
          <Icon name="envelope" size={40} />
        </span>
        <p className="hand gb-thanks-title">收到啦{state.name ? `，${state.name}` : ''}！</p>
        <p className="muted">
          {state.approved
            ? '你的祝福已经贴到下面的墙上了，谢谢你 ♡'
            : '我们会一张一张认真读，然后把它贴到祝福墙上 ♡'}
        </p>
      </div>
    );
  }

  return (
    <form action={action} className="gb-form" key={state.status === 'error' ? state.message : 'form'}>
      <label className="field">
        <span>你的名字</span>
        <input
          name="name"
          required
          maxLength={24}
          autoComplete="name"
          placeholder="我们该怎么称呼你"
          defaultValue={state.values?.name}
        />
      </label>

      {rsvp ? (
        <fieldset className="gb-rsvp">
          <legend className="field-label">
            能来参加婚礼吗？{deadline ? <span className="faint">（请在 {deadline} 前告诉我们）</span> : null}
          </legend>
          <div className="gb-choices">
            {[
              ['yes', '一定到场'],
              ['maybe', '还不确定'],
              ['no', '遗憾缺席'],
            ].map(([value, label]) => (
              <label key={value} className={`gb-choice ${attending === value ? 'on' : ''}`}>
                <input
                  type="radio"
                  name="attending"
                  value={value}
                  checked={attending === value}
                  onChange={() => setAttending(value)}
                />
                {label}
              </label>
            ))}
          </div>
          {attending === 'yes' || attending === 'maybe' ? (
            <div className="gb-row">
              <label className="field">
                <span>一共几位</span>
                <select name="partySize" defaultValue={state.values?.partySize ?? '1'}>
                  {Array.from({ length: 10 }, (_, i) => (
                    <option key={i + 1} value={i + 1}>
                      {i + 1} 位
                    </option>
                  ))}
                </select>
              </label>
              <label className="field">
                <span>联系方式（只有我们看得到）</span>
                <input
                  name="contact"
                  maxLength={40}
                  placeholder="手机 / 微信，方便我们安排"
                  defaultValue={state.values?.contact}
                />
              </label>
            </div>
          ) : null}
        </fieldset>
      ) : null}

      <label className="field">
        <span>写给我们的话</span>
        <textarea
          name="message"
          maxLength={300}
          rows={4}
          placeholder="一句祝福、一段回忆，或者只是一个笑脸～"
          defaultValue={state.values?.message}
        />
      </label>

      <label className="gb-honey" aria-hidden>
        请不要填写这一项 <input name="website" tabIndex={-1} autoComplete="off" />
      </label>

      {state.status === 'error' ? <p className="notice notice-red">{state.message}</p> : null}

      <button type="submit" className="btn btn-red gb-submit" disabled={pending}>
        <Icon name="envelope" size={18} />
        {pending ? '正在寄出…' : '寄出'}
      </button>
    </form>
  );
}
