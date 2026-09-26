'use server';

import { revalidatePath } from 'next/cache';
import { redirect } from 'next/navigation';
import { assertSession } from '@/lib/auth';
import { isDay } from '@/lib/dates';
import { deleteNote, setNoteApproved } from '@/lib/guests';
import { MEDIA_KEY, PHOTO_KEY, newId } from '@/lib/ids';
import { deleteMoment, getMoment, insertMoment, isMomentKind, updateMoment, type MomentInput } from '@/lib/moments';
import { deletePhotos, insertPhoto, updatePhotos, type PhotoPatch } from '@/lib/photos';
import { DEFAULT_SETTINGS, getSettings, saveSettings } from '@/lib/settings';
import { deleteKeys } from '@/lib/storage';
import { clearDemo, loadDemo } from '@/lib/demo';
import type { Partner, Settings, Visibility } from '@/lib/types';

const text = (value: FormDataEntryValue | null | undefined, max = 2000) => {
  const v = String(value ?? '').replace(/\r\n/g, '\n').trim().slice(0, max);
  return v || null;
};
const visibilityOf = (value: unknown): Visibility => (value === 'public' ? 'public' : 'private');
const partnerOf = (value: unknown): Partner | null => (value === 'a' || value === 'b' ? value : null);
const TAKEN_AT = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$/;

function refreshEverything() {
  revalidatePath('/', 'layout');
}

/* ------------------------------------------------------------------ photos */

export type NewPhotoInput = {
  id: string;
  lgKey: string;
  smKey: string;
  origKey: string | null;
  width: number;
  height: number;
  blurData: string | null;
  color: string | null;
  takenAt: string | null;
  latitude: number | null;
  longitude: number | null;
  camera: string | null;
  caption: string | null;
  place: string | null;
  momentId: string | null;
  visibility: Visibility;
};

/** Called by the uploader once the files are in storage. */
export async function createPhoto(input: NewPhotoInput): Promise<{ ok: true } | { ok: false; error: string }> {
  const session = await assertSession();
  const prefix = `photos/${input.id}-`;
  const keyOk = (key: string | null, variant: string) =>
    key !== null && PHOTO_KEY.test(key) && key.startsWith(prefix) && key.includes(`-${variant}.`);
  if (!/^[a-z0-9]{12}$/.test(input.id) || !keyOk(input.lgKey, 'lg') || !keyOk(input.smKey, 'sm')) {
    return { ok: false, error: '文件路径不对' };
  }
  if (input.origKey !== null && !keyOk(input.origKey, 'orig')) return { ok: false, error: '原图路径不对' };
  const width = Math.round(Number(input.width));
  const height = Math.round(Number(input.height));
  if (!(width > 0 && height > 0 && width < 20000 && height < 20000)) return { ok: false, error: '尺寸不对' };
  const latitude = Number.isFinite(input.latitude) && Math.abs(input.latitude!) <= 90 ? input.latitude : null;
  const longitude = Number.isFinite(input.longitude) && Math.abs(input.longitude!) <= 180 ? input.longitude : null;
  const momentId = input.momentId && (await getMoment(input.momentId)) ? input.momentId : null;

  await insertPhoto({
    id: input.id,
    lgKey: input.lgKey,
    smKey: input.smKey,
    origKey: input.origKey,
    width,
    height,
    blurData: input.blurData?.startsWith('data:image/jpeg;base64,') && input.blurData.length < 6000 ? input.blurData : null,
    color: input.color && /^#[0-9a-f]{6}$/i.test(input.color) ? input.color : null,
    caption: text(input.caption, 60),
    note: null,
    takenAt: input.takenAt && TAKEN_AT.test(input.takenAt) ? input.takenAt.slice(0, 19).padEnd(19, ':00') : null,
    place: text(input.place, 60),
    latitude,
    longitude,
    camera: text(input.camera, 80),
    author: session.partner,
    momentId,
    visibility: visibilityOf(input.visibility),
  });
  refreshEverything();
  return { ok: true };
}

export async function updatePhotoAction(id: string, form: FormData) {
  await assertSession();
  const takenRaw = String(form.get('takenAt') ?? '').trim();
  const momentId = text(form.get('momentId'), 20);
  await updatePhotos([id], {
    caption: text(form.get('caption'), 60),
    note: text(form.get('note'), 600),
    place: text(form.get('place'), 60),
    takenAt: TAKEN_AT.test(takenRaw) ? (takenRaw.length === 16 ? `${takenRaw}:00` : takenRaw) : null,
    author: partnerOf(form.get('author')),
    momentId: momentId && (await getMoment(momentId)) ? momentId : null,
    visibility: visibilityOf(form.get('visibility')),
    featured: form.get('featured') === 'on',
    favorite: form.get('favorite') === 'on',
  });
  refreshEverything();
  const back = String(form.get('back') ?? '');
  redirect(back.startsWith('/admin/') ? back : '/admin/photos');
}

export type BulkOp =
  | 'public'
  | 'private'
  | 'feature'
  | 'unfeature'
  | 'favorite'
  | 'unfavorite'
  | 'moment'
  | 'delete';

export async function bulkPhotos(ids: string[], op: BulkOp, momentId?: string | null) {
  await assertSession();
  const clean = ids.filter((id) => /^[a-z0-9-]{1,40}$/.test(id)).slice(0, 500);
  if (clean.length === 0) return;
  const patches: Record<Exclude<BulkOp, 'delete' | 'moment'>, PhotoPatch> = {
    public: { visibility: 'public' },
    private: { visibility: 'private' },
    feature: { featured: true },
    unfeature: { featured: false },
    favorite: { favorite: true },
    unfavorite: { favorite: false },
  };
  if (op === 'delete') {
    const removed = await deletePhotos(clean);
    await deleteKeys(removed.flatMap((p) => [p.lgKey, p.smKey, p.origKey])).catch((error) =>
      console.error('Could not delete files from storage', error),
    );
  } else if (op === 'moment') {
    const target = momentId && (await getMoment(momentId)) ? momentId : null;
    await updatePhotos(clean, { momentId: target });
  } else if (patches[op]) {
    await updatePhotos(clean, patches[op]);
  }
  refreshEverything();
}

export async function deletePhotoAction(id: string) {
  await bulkPhotos([id], 'delete');
  redirect('/admin/photos');
}

/* ----------------------------------------------------------------- moments */

export async function saveMomentAction(id: string | null, form: FormData) {
  await assertSession();
  const startsOn = String(form.get('startsOn') ?? '');
  const endsOn = String(form.get('endsOn') ?? '');
  const kind = form.get('kind');
  const input: MomentInput = {
    title: text(form.get('title'), 60) ?? '未命名的回忆',
    kind: isMomentKind(kind) ? kind : 'date',
    startsOn: isDay(startsOn) ? startsOn : new Date().toISOString().slice(0, 10),
    endsOn: isDay(endsOn) && endsOn > startsOn ? endsOn : null,
    place: text(form.get('place'), 60),
    story: text(form.get('story'), 3000),
    coverPhotoId: text(form.get('coverPhotoId'), 40),
    visibility: visibilityOf(form.get('visibility')),
  };
  let momentId = id;
  if (momentId) {
    await updateMoment(momentId, input);
  } else {
    momentId = newId();
    await insertMoment(momentId, input);
  }
  const photoIds = String(form.get('photoIds') ?? '')
    .split(',')
    .filter((pid) => /^[a-z0-9-]{1,40}$/.test(pid));
  if (photoIds.length) await updatePhotos(photoIds, { momentId });
  refreshEverything();
  redirect(`/us/moments/${momentId}`);
}

export async function deleteMomentAction(id: string) {
  await assertSession();
  await deleteMoment(id);
  refreshEverything();
  redirect('/admin/moments');
}

/* ---------------------------------------------------------------- settings */

export async function saveSettingsAction(form: FormData) {
  await assertSession();
  const current = await getSettings();
  const next: Partial<Settings> = {};
  for (const key of Object.keys(DEFAULT_SETTINGS) as (keyof Settings)[]) {
    const fallback = DEFAULT_SETTINGS[key];
    if (typeof fallback === 'boolean') {
      (next as Record<string, unknown>)[key] = form.get(key) === 'on';
    } else if (form.has(key)) {
      (next as Record<string, unknown>)[key] = String(form.get(key) ?? '').replace(/\r\n/g, '\n').trim().slice(0, 2000);
    }
  }
  for (const key of ['togetherSince', 'firstMet', 'weddingDate', 'rsvpDeadline'] as const) {
    if (next[key] && !isDay(next[key])) next[key] = current[key];
  }
  if (next.weddingTime && !/^\d{2}:\d{2}$/.test(next.weddingTime)) next.weddingTime = current.weddingTime;
  if (next.music && !/^https?:\/\//.test(next.music) && !MEDIA_KEY.test(next.music)) next.music = current.music;
  await saveSettings(next);
  refreshEverything();
  redirect('/admin/settings?saved=1');
}

/* ------------------------------------------------------------------ guests */

export async function guestNoteAction(form: FormData) {
  await assertSession();
  const id = String(form.get('id') ?? '');
  const op = String(form.get('op') ?? '');
  if (!/^[a-z0-9]{12}$/.test(id)) return;
  if (op === 'approve') await setNoteApproved(id, true);
  else if (op === 'hide') await setNoteApproved(id, false);
  else if (op === 'delete') await deleteNote(id);
  refreshEverything();
}

/* -------------------------------------------------------------------- demo */

export async function loadDemoAction() {
  await assertSession();
  await loadDemo();
  refreshEverything();
  redirect('/admin/photos?demo=1');
}

export async function clearDemoAction() {
  await assertSession();
  await clearDemo();
  refreshEverything();
  redirect('/admin/photos');
}

