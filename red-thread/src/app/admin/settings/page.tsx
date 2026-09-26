import { requireSession } from '@/lib/auth';
import { getSettings } from '@/lib/settings';
import { urlFor } from '@/lib/storage';
import { saveSettingsAction } from '@/app/admin/actions';
import { SaveButton } from '@/components/admin/Small';
import { MusicField } from '@/components/admin/MusicField';

export const dynamic = 'force-dynamic';

export default async function SettingsPage({ searchParams }: { searchParams: Promise<{ saved?: string }> }) {
  await requireSession('/admin/settings');
  const [{ saved }, s] = await Promise.all([searchParams, getSettings()]);
  const musicPreview = s.music ? await urlFor(s.music, false) : null;
  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>我们 & 请柬</h1>
          <p className="muted small">这些内容会出现在公开相册、信封和婚礼请柬里。</p>
        </div>
      </header>
      {saved ? <p className="notice saved-note">保存好啦 ♡</p> : null}

      <form action={saveSettingsAction} className="settings">
        <fieldset className="card form-grid">
          <legend>我们</legend>
          <label className="field">
            <span>名字 · 用 ADMIN_EMAIL 登录的人</span>
            <input name="partnerA" defaultValue={s.partnerA} maxLength={20} required />
          </label>
          <label className="field">
            <span>名字 · 用 PARTNER_EMAIL 登录的人</span>
            <input name="partnerB" defaultValue={s.partnerB} maxLength={20} required />
          </label>
          <label className="field">
            <span>在一起的日子</span>
            <input type="date" name="togetherSince" defaultValue={s.togetherSince} />
          </label>
          <label className="field">
            <span>第一次见面（可选）</span>
            <input type="date" name="firstMet" defaultValue={s.firstMet} />
          </label>
          <label className="field">
            <span>火漆印上的字</span>
            <input name="initials" defaultValue={s.initials} maxLength={8} />
            <small>两个名字的首字母最好看，比如 C & M</small>
          </label>
          <label className="field">
            <span>网站标题（可选）</span>
            <input name="siteTitle" defaultValue={s.siteTitle} maxLength={40} placeholder={`${s.partnerA} & ${s.partnerB} 的朝朝暮暮`} />
          </label>
          <label className="field span-2">
            <span>一句话</span>
            <input name="tagline" defaultValue={s.tagline} maxLength={60} />
          </label>
          <label className="field span-2">
            <span>故事的开头</span>
            <textarea name="intro" defaultValue={s.intro} rows={3} maxLength={400} />
          </label>
          <label className="field span-2">
            <span>结尾的那句话</span>
            <input name="closingLine" defaultValue={s.closingLine} maxLength={30} />
          </label>
        </fieldset>

        <fieldset className="card form-grid">
          <legend>信封</legend>
          <label className="check span-2">
            <input type="checkbox" name="envelopeEnabled" defaultChecked={s.envelopeEnabled} />
            客人打开相册时，先看到一封盖着火漆的信
          </label>
          <label className="field span-2">
            <span>信封上的字（没有专属链接时显示）</span>
            <input name="envelopeLine" defaultValue={s.envelopeLine} maxLength={30} />
            <small>用“分享”页生成的专属链接打开时，信封上会写着“致 某某”。</small>
          </label>
          <MusicField initial={s.music} previewUrl={musicPreview} />
        </fieldset>

        <fieldset className="card form-grid">
          <legend>婚礼</legend>
          <label className="check span-2">
            <input type="checkbox" name="weddingEnabled" defaultChecked={s.weddingEnabled} />
            在公开相册里放上婚礼请柬（还没定日子就先关着）
          </label>
          <label className="field">
            <span>日期</span>
            <input type="date" name="weddingDate" defaultValue={s.weddingDate} />
          </label>
          <label className="field">
            <span>时间</span>
            <input type="time" name="weddingTime" defaultValue={s.weddingTime} />
          </label>
          <label className="field">
            <span>场地</span>
            <input name="weddingVenue" defaultValue={s.weddingVenue} maxLength={60} placeholder="某某酒店 · 三楼宴会厅" />
          </label>
          <label className="field">
            <span>地址</span>
            <input name="weddingAddress" defaultValue={s.weddingAddress} maxLength={120} />
          </label>
          <label className="field">
            <span>坐标（可选，经度,纬度）</span>
            <input name="weddingLngLat" defaultValue={s.weddingLngLat} placeholder="120.155,30.274" />
            <small>在高德地图里找到场地后复制坐标，导航会更准。</small>
          </label>
          <label className="field">
            <span>着装建议（可选）</span>
            <input name="dressCode" defaultValue={s.dressCode} maxLength={60} />
          </label>
          <label className="field span-2">
            <span>请柬上的话</span>
            <textarea name="weddingInvitation" defaultValue={s.weddingInvitation} rows={3} maxLength={400} />
          </label>
          <label className="field span-2">
            <span>流程（一行一项，时间在前）</span>
            <textarea name="weddingSchedule" defaultValue={s.weddingSchedule} rows={4} maxLength={400} />
          </label>
        </fieldset>

        <fieldset className="card form-grid">
          <legend>回执与祝福</legend>
          <label className="check span-2">
            <input type="checkbox" name="rsvpEnabled" defaultChecked={s.rsvpEnabled} />
            请柬里可以回复是否出席、几位（开启婚礼请柬时生效）
          </label>
          <label className="field">
            <span>回复截止（可选）</span>
            <input type="date" name="rsvpDeadline" defaultValue={s.rsvpDeadline} />
          </label>
          <label className="check">
            <input type="checkbox" name="autoApproveNotes" defaultChecked={s.autoApproveNotes} />
            祝福不用审核，直接贴到祝福墙上
          </label>
        </fieldset>

        <div className="settings-save">
          <SaveButton />
        </div>
      </form>
    </div>
  );
}
