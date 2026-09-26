'use server';

import { createHash } from 'node:crypto';
import { headers } from 'next/headers';
import { revalidatePath } from 'next/cache';
import { insertNote, recentNotesFrom } from '@/lib/guests';
import { newId } from '@/lib/ids';
import { getSettings } from '@/lib/settings';
import type { Attendance } from '@/lib/types';

export type GuestFormState = {
  status: 'idle' | 'ok' | 'error';
  message?: string;
  approved?: boolean;
  name?: string;
  /** What the guest typed, so a rejected note can be fixed instead of rewritten. */
  values?: { name: string; message: string; contact: string; partySize: string };
};

const text = (form: FormData, key: string, max: number) =>
  String(form.get(key) ?? '')
    .replace(/\s+\n/g, '\n')
    .trim()
    .slice(0, max);

export async function submitGuestNote(_prev: GuestFormState, form: FormData): Promise<GuestFormState> {
  // Bots love filling every field, including the one people never see.
  if (text(form, 'website', 100)) return { status: 'ok', approved: false };

  const settings = await getSettings();
  const name = text(form, 'name', 24);
  const message = text(form, 'message', 300);
  const contact = text(form, 'contact', 40);
  const rsvp = settings.weddingEnabled && settings.rsvpEnabled;
  const attendingRaw = String(form.get('attending') ?? '');
  const attending: Attendance | null =
    rsvp && (attendingRaw === 'yes' || attendingRaw === 'no' || attendingRaw === 'maybe') ? attendingRaw : null;
  const partySizeRaw = Number(form.get('partySize') ?? 1);
  const partySize =
    attending === 'yes' || attending === 'maybe'
      ? Math.min(Math.max(Number.isFinite(partySizeRaw) ? Math.round(partySizeRaw) : 1, 1), 10)
      : null;

  const values = { name, message, contact, partySize: String(partySize ?? 1) };
  if (!name) return { status: 'error', message: '留下你的名字吧，我们想知道是谁送来的祝福～', values };
  if (!message && !attending) return { status: 'error', message: '写一句祝福，或者告诉我们你是否能来～', values };

  const head = await headers();
  const ip = (head.get('x-forwarded-for') ?? head.get('x-real-ip') ?? 'unknown').split(',')[0].trim();
  const ipHash = createHash('sha256')
    .update(`${ip}:${process.env.AUTH_SECRET ?? 'red-thread'}`)
    .digest('hex')
    .slice(0, 24);
  if ((await recentNotesFrom(ipHash, 10)) >= 5) {
    return { status: 'error', message: '你的祝福我们都收到啦，歇一会儿再写吧 :)', values };
  }

  const approved = settings.autoApproveNotes;
  await insertNote({
    id: newId(),
    name,
    message: message || null,
    attending,
    partySize,
    contact: contact || null,
    approved,
    ipHash,
  });
  if (approved) revalidatePath('/');
  return { status: 'ok', approved, name };
}
