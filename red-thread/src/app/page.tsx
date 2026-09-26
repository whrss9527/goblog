import type { Metadata } from 'next';
import Link from 'next/link';
import type { CSSProperties } from 'react';
import { loadPublicAlbum } from '@/lib/album';
import { getSession } from '@/lib/auth';
import { dayNumber, formatDay, formatRange, isDay, today } from '@/lib/dates';
import { MOMENT_KINDS } from '@/lib/moments';
import { siteTitle } from '@/lib/settings';
import { urlFor } from '@/lib/storage';
import { seeded } from '@/lib/ui';
import { mapLinks, weddingDateParts, weddingReady, weddingSchedule } from '@/lib/wedding';
import { Icon, KIND_ICON } from '@/components/Icon';
import { LightboxProvider } from '@/components/Lightbox';
import { Corkboard, HeroFan, PolaroidCluster } from '@/components/PhotoGroups';
import { DaysCounter, MusicPlayer, Petals, RevealObserver, WeddingCountdown } from '@/components/Ambient';
import { Envelope } from '@/components/Envelope';
import { RedThread } from '@/components/RedThread';
import { GuestbookForm } from '@/components/Guestbook';
import { Bow, Stamp } from '@/components/Bits';

export const dynamic = 'force-dynamic';

type Props = { searchParams: Promise<Record<string, string | string[] | undefined>> };

const guestName = (value: string | string[] | undefined) =>
  (Array.isArray(value) ? value[0] : value)?.replace(/[<>\n\r\t]/g, '').trim().slice(0, 20) || null;

export async function generateMetadata(): Promise<Metadata> {
  const album = await loadPublicAlbum();
  const { settings: s } = album;
  const title = siteTitle(s);
  const cover = album.featured[0]?.src;
  const description = weddingReady(s)
    ? `${s.partnerA} & ${s.partnerB} 的婚礼 · ${formatDay(s.weddingDate)}。${s.tagline}`
    : s.tagline;
  return {
    title,
    description,
    openGraph: { title, description, images: cover ? [cover] : undefined, type: 'website' },
    twitter: { card: cover ? 'summary_large_image' : 'summary', title, description },
  };
}

const NOTE_COLORS = ['#fdf1c8', '#fbe0e0', '#e3efd9', '#e0e9f5', '#f3e2f0', '#fde6cf'];

export default async function Home({ searchParams }: Props) {
  const [album, session, params] = await Promise.all([loadPublicAlbum(), getSession(), searchParams]);
  const { settings: s, chapters, loose, featured, stamps, blessings, stats } = album;
  const to = guestName(params.to);
  const wedding = weddingReady(s);
  const date = wedding ? weddingDateParts(s) : null;
  const since = isDay(s.togetherSince) ? s.togetherSince : today();
  const music = s.music ? await urlFor(s.music, false) : null;
  const rsvp = wedding && s.rsvpEnabled;
  const kindLabel = Object.fromEntries(MOMENT_KINDS.map((k) => [k.value, k.label]));

  return (
    <LightboxProvider>
      {s.envelopeEnabled ? (
        <Envelope
          to={to}
          line={s.envelopeLine}
          initials={s.initials}
          names={`${s.partnerA} & ${s.partnerB}`}
          letterTop={wedding ? 'Save the Date' : 'Our Love Story'}
          letterBottom={wedding && date ? date.dot : `since ${formatDay(since, 'dot')}`}
        />
      ) : null}
      <Petals />
      <RevealObserver />
      {session ? (
        <Link href="/us" className="us-pill">
          <Icon name="home" size={15} /> 回到我们的小窝
        </Link>
      ) : null}

      <header className="hero">
        <p className="hero-kicker">{wedding ? 'Save the Date' : 'Our Love Story'}</p>
        <h1 className="hero-names">
          <span>{s.partnerA}</span>
          <span className="amp script">&amp;</span>
          <span>{s.partnerB}</span>
        </h1>
        {s.tagline ? <p className="hero-tagline hand">{s.tagline}</p> : null}
        {featured.length > 0 ? (
          <HeroFan photos={featured} />
        ) : (
          <div className="fan-empty hand">这里会放上我们最喜欢的几张照片</div>
        )}
        <div className="hero-bottom">
          <DaysCounter since={since} initialDay={dayNumber(since, today())} />
          {wedding && date ? (
            <div className="hero-date">
              <p className="hero-date-label">我们结婚啦</p>
              <p className="hero-date-line">{date.dot}</p>
              <p className="hero-date-sub">
                {date.weekday} · {s.weddingTime}
              </p>
              <WeddingCountdown date={s.weddingDate} time={s.weddingTime} />
            </div>
          ) : null}
        </div>
        <a href="#story" className="scroll-cue">
          <span className="hand">顺着红线往下走</span>
          <Icon name="down" size={18} />
        </a>
      </header>

      <main>
        <section className="story" id="story">
          <RedThread />
          <header className="story-intro" data-reveal>
            <span className="knot knot-heart" data-knot>
              <Icon name="heart" size={28} />
            </span>
            <div className="story-intro-text">
              <p className="section-kicker">Our Story</p>
              <h2 className="section-title">我们的故事</h2>
              {s.intro ? <p>{s.intro}</p> : null}
            </div>
          </header>

          {chapters.map(({ moment, photos, sealed }) => {
            const special = moment.kind === 'meet' || moment.kind === 'anniversary' || moment.kind === 'milestone';
            return (
              <article key={moment.id} className={`chapter ${special ? 'milestone' : ''}`} data-reveal>
                <span className={`knot ${special ? 'knot-heart' : ''}`} data-knot>
                  {special ? <Icon name="heart" size={28} /> : null}
                  <span className="knot-date">{formatDay(moment.startsOn, 'dot')}</span>
                </span>
                <div className="chapter-text">
                  <span className="chapter-kind">
                    <Icon name={KIND_ICON[moment.kind]} size={14} />
                    {kindLabel[moment.kind]}
                  </span>
                  <h3 className="chapter-title">{moment.title}</h3>
                  <p className="chapter-meta">
                    <span>
                      <Icon name="calendar" size={14} />
                      {formatRange(moment.startsOn, moment.endsOn)}
                    </span>
                    {moment.place ? (
                      <span>
                        <Icon name="pin" size={14} />
                        {moment.place}
                      </span>
                    ) : null}
                  </p>
                  {moment.story ? <p className="chapter-story">{moment.story}</p> : null}
                  {sealed > 0 ? (
                    <p className="chapter-sealed">
                      <Icon name="lock" size={15} />
                      还有 {sealed} 张，只给彼此看
                    </p>
                  ) : null}
                </div>
                {photos.length > 0 ? (
                  <div className="chapter-photos">
                    <PolaroidCluster photos={photos} label={moment.title} />
                  </div>
                ) : null}
              </article>
            );
          })}

          <div className="story-end" data-reveal>
            <span className="knot knot-heart" data-knot>
              <Icon name="heart" size={28} />
            </span>
            <div className="story-end-text">
              {chapters.length === 0 ? (
                <p className="hand">故事才刚刚开始写呢</p>
              ) : wedding ? (
                <>
                  <p className="hand">下一个结，想请你一起来系</p>
                  <a href="#wedding" className="btn btn-red">
                    <Icon name="ring" size={18} /> 看看婚礼请柬
                  </a>
                </>
              ) : (
                <p className="hand">未完，待续……</p>
              )}
            </div>
          </div>
        </section>

        {loose.length > 0 ? (
          <section className="section">
            <div className="section-head" data-reveal>
              <p className="section-kicker">Little Things</p>
              <h2 className="section-title">零零碎碎的日常</h2>
              <p className="section-sub">没有被写进章节的那些瞬间，也一样重要。</p>
            </div>
            <div data-reveal>
              <Corkboard photos={loose} />
            </div>
          </section>
        ) : null}

        {stamps.length > 0 ? (
          <section className="section">
            <div className="section-head" data-reveal>
              <p className="section-kicker">Places</p>
              <h2 className="section-title">一起去过的地方</h2>
              <p className="section-sub">每一枚邮戳，都是一段一起出发的路。</p>
            </div>
            <div className="stamps" data-reveal>
              {stamps.map((stamp) => (
                <Stamp
                  key={stamp.place}
                  place={stamp.place}
                  date={stamp.day ? formatDay(stamp.day, 'dot').slice(0, 7) : '♡'}
                  thumb={stamp.thumb}
                />
              ))}
            </div>
          </section>
        ) : null}

        <section className="section">
          <div className="section-head" data-reveal>
            <p className="section-kicker">In Numbers</p>
            <h2 className="section-title">我们的数字</h2>
          </div>
          <div className="numbers" data-reveal>
            <div className="number">
              <b>{stats.days.toLocaleString('zh-CN')}</b>
              <span>天，在一起</span>
            </div>
            <div className="number">
              <b>{stats.weekends.toLocaleString('zh-CN')}</b>
              <span>个周末，一起过</span>
            </div>
            {stats.places > 0 ? (
              <div className="number">
                <b>{stats.places}</b>
                <span>个地方，一起去过</span>
              </div>
            ) : null}
            {stats.photos > 0 ? (
              <div className="number">
                <b>{stats.photos.toLocaleString('zh-CN')}</b>
                <span>{stats.hidden > 0 ? `张照片，${stats.hidden} 张悄悄藏着` : '张照片，都在这里'}</span>
              </div>
            ) : null}
            {stats.valentines > 0 ? (
              <div className="number">
                <b>{stats.valentines}</b>
                <span>个情人节，都是你</span>
              </div>
            ) : null}
          </div>
        </section>

        {wedding && date ? (
          <section className="section" id="wedding">
            <div className="invite" data-reveal>
              <span className="xi" aria-hidden>
                囍
              </span>
              <p className="invite-kicker">Wedding Invitation</p>
              {to ? <p className="invite-to">诚挚邀请 {to}</p> : null}
              <p className="invite-words">{s.weddingInvitation}</p>
              <div className="invite-date">
                <span>
                  {date.year} 年 {date.month} 月
                </span>
                <span className="invite-day">{String(date.day).padStart(2, '0')}</span>
                <span>
                  {date.weekday} {s.weddingTime}
                </span>
              </div>
              {s.weddingVenue || s.weddingAddress ? (
                <div>
                  {s.weddingVenue ? <p className="invite-venue">{s.weddingVenue}</p> : null}
                  {s.weddingAddress ? <p className="invite-address">{s.weddingAddress}</p> : null}
                </div>
              ) : null}
              <div className="invite-actions">
                {mapLinks(s).map((link) => (
                  <a key={link.label} className="btn btn-sm" href={link.href} target="_blank" rel="noreferrer">
                    <Icon name="map" size={15} /> {link.label}
                  </a>
                ))}
                <a className="btn btn-sm" href="/wedding.ics" download>
                  <Icon name="calendar" size={15} /> 加入日历
                </a>
              </div>
              {weddingSchedule(s).length > 0 ? (
                <ol className="invite-schedule">
                  {weddingSchedule(s).map((item, i) => (
                    <li key={i}>
                      <time>{item.time}</time>
                      <span>{item.what}</span>
                    </li>
                  ))}
                </ol>
              ) : null}
              {s.dressCode ? <p className="invite-dress">着装建议：{s.dressCode}</p> : null}
              <WeddingCountdown date={s.weddingDate} time={s.weddingTime} />
            </div>
          </section>
        ) : null}

        <section className="section" id="guestbook">
          <div className="section-head" data-reveal>
            <p className="section-kicker">{rsvp ? 'RSVP' : 'Guestbook'}</p>
            <h2 className="section-title">{rsvp ? '回执与祝福' : '写给我们'}</h2>
            <p className="section-sub">
              {rsvp ? '告诉我们你能不能来，顺便留一句话给我们吧。' : '留一句话给我们，我们会好好收着。'}
            </p>
          </div>
          <div className="gb-paper" data-reveal>
            <GuestbookForm rsvp={rsvp} deadline={isDay(s.rsvpDeadline) ? formatDay(s.rsvpDeadline) : null} />
          </div>
          {blessings.length > 0 ? (
            <div className="wall">
              {blessings.map((note) => (
                <article
                  key={note.id}
                  className="note"
                  data-reveal
                  style={
                    {
                      '--tilt': `${(seeded(note.id) - 0.5) * 6}deg`,
                      '--note': NOTE_COLORS[Math.floor(seeded(note.id, 2) * NOTE_COLORS.length)],
                    } as CSSProperties
                  }
                >
                  <span className="tape" />
                  <p>{note.message}</p>
                  <footer>— {note.name}</footer>
                </article>
              ))}
            </div>
          ) : (
            <p className="wall-empty">第一个写下祝福的人，会被我们记很久很久。</p>
          )}
        </section>
      </main>

      <footer className="closing">
        <Bow />
        <p className="closing-line">{s.closingLine}</p>
        <p className="closing-names">
          {s.partnerA} &amp; {s.partnerB}
        </p>
        <p className="closing-since">SINCE {formatDay(since, 'dot')}</p>
        <p className="closing-fine">
          <Link href="/login" aria-label="我们的入口">
            ♡
          </Link>
        </p>
      </footer>

      {music ? <MusicPlayer src={music} autoplayInWeChat={!s.envelopeEnabled} /> : null}
    </LightboxProvider>
  );
}
