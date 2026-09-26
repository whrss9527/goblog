import { requireSession } from '@/lib/auth';
import { TIMEZONE } from '@/lib/dates';
import { listNotes } from '@/lib/guests';
import { guestNoteAction } from '@/app/admin/actions';
import { Icon } from '@/components/Icon';
import type { GuestNote } from '@/lib/types';

export const dynamic = 'force-dynamic';

const ATTENDING: Record<string, string> = { yes: '一定到场', maybe: '还不确定', no: '遗憾缺席' };

function summary(notes: GuestNote[]) {
  const count = (a: string) => notes.filter((n) => n.attending === a);
  const people = (list: GuestNote[]) => list.reduce((sum, n) => sum + (n.partySize ?? 1), 0);
  return {
    yes: count('yes').length,
    yesPeople: people(count('yes')),
    maybe: count('maybe').length,
    maybePeople: people(count('maybe')),
    no: count('no').length,
    pending: notes.filter((n) => !n.approved && n.message).length,
  };
}

const time = (iso: string) =>
  new Intl.DateTimeFormat('zh-CN', {
    timeZone: TIMEZONE,
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));

export default async function GuestsPage() {
  await requireSession('/admin/guests');
  const notes = await listNotes();
  const s = summary(notes);
  return (
    <div>
      <header className="admin-head">
        <div>
          <h1>回执与祝福</h1>
          <p className="muted small">客人在公开相册里留下的话。祝福要点“贴上墙”才会公开，联系方式永远只有我们看得到。</p>
        </div>
        <a href="/admin/guests/export" className="btn btn-sm">
          <Icon name="download" size={15} /> 导出表格
        </a>
      </header>

      <div className="stat-row">
        <div className="stat card">
          <b>{s.yesPeople}</b>
          <span>位会来（{s.yes} 份回执）</span>
        </div>
        <div className="stat card">
          <b>{s.maybePeople}</b>
          <span>位还不确定</span>
        </div>
        <div className="stat card">
          <b>{s.no}</b>
          <span>份遗憾缺席</span>
        </div>
        <div className="stat card">
          <b>{s.pending}</b>
          <span>条祝福等着上墙</span>
        </div>
      </div>

      {notes.length === 0 ? (
        <div className="empty card">
          <p className="hand">还没有收到回信</p>
          <p>把专属链接发给朋友们吧（在“分享”里生成）。</p>
        </div>
      ) : (
        <ul className="guest-list">
          {notes.map((note) => (
            <li key={note.id} className={`card guest ${note.approved ? 'approved' : ''}`}>
              <div className="guest-head">
                <strong>{note.name}</strong>
                {note.attending ? (
                  <span className={`row-tag att-${note.attending}`}>
                    {ATTENDING[note.attending]}
                    {note.partySize && note.attending !== 'no' ? ` · ${note.partySize} 位` : ''}
                  </span>
                ) : null}
                <span className="faint small">{time(note.createdAt)}</span>
              </div>
              {note.message ? <p className="guest-message hand">{note.message}</p> : null}
              {note.contact ? (
                <p className="small muted">
                  <Icon name="lock" size={12} /> {note.contact}
                </p>
              ) : null}
              <form action={guestNoteAction} className="guest-actions">
                <input type="hidden" name="id" value={note.id} />
                {note.message ? (
                  note.approved ? (
                    <button name="op" value="hide" className="btn btn-sm">
                      从墙上取下
                    </button>
                  ) : (
                    <button name="op" value="approve" className="btn btn-sm btn-red">
                      <Icon name="check" size={15} /> 贴上墙
                    </button>
                  )
                ) : null}
                <button name="op" value="delete" className="btn btn-sm btn-ghost">
                  <Icon name="trash" size={15} /> 删除
                </button>
              </form>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
