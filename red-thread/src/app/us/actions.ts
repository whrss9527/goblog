'use server';

import { assertSession } from '@/lib/auth';
import { randomPhoto, toCards } from '@/lib/photos';
import { getSettings } from '@/lib/settings';
import type { PhotoCard } from '@/lib/types';

/** 抽一张回忆: any photo at all, private ones included. */
export async function drawMemory(): Promise<PhotoCard | null> {
  await assertSession();
  const photo = await randomPhoto();
  if (!photo) return null;
  const [card] = await toCards([photo], await getSettings(), { forUs: true });
  return card;
}
