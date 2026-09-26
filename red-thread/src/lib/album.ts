import 'server-only';
import { cache } from 'react';
import { countPhotos, listPhotos, privateCountsByMoment, toCards } from './photos';
import { listMoments } from './moments';
import { publicBlessings } from './guests';
import { getSettings } from './settings';
import { dayNumber, isDay, occurrences, today } from './dates';
import type { GuestNote, Moment, PhotoCard, Settings } from './types';

export type Chapter = { moment: Moment; photos: PhotoCard[]; sealed: number };
export type Stamp = { place: string; day: string | null; thumb: string | null };

export type PublicAlbum = {
  settings: Settings;
  chapters: Chapter[];
  loose: PhotoCard[];
  featured: PhotoCard[];
  stamps: Stamp[];
  blessings: GuestNote[];
  stats: { days: number; weekends: number; places: number; photos: number; hidden: number; valentines: number };
};

/** Everything a guest may see, and nothing more. Deduplicated per request. */
export const loadPublicAlbum = cache(async (): Promise<PublicAlbum> => {
  const settings = await getSettings();
  const [moments, photos, sealedCounts, blessings, counts] = await Promise.all([
    listMoments({ publicOnly: true }),
    listPhotos({ visibility: 'public', order: 'oldest' }),
    privateCountsByMoment(),
    publicBlessings(),
    countPhotos(),
  ]);
  const cards = await toCards(photos, settings);
  const cardById = new Map(cards.map((card) => [card.id, card]));
  const publicMomentIds = new Set(moments.map((moment) => moment.id));

  const chapters: Chapter[] = moments.map((moment) => {
    const own = photos.filter((photo) => photo.momentId === moment.id).map((photo) => cardById.get(photo.id)!);
    // The chosen cover leads the little pile of polaroids.
    const coverIndex = own.findIndex((card) => card.id === moment.coverPhotoId);
    if (coverIndex > 0) own.unshift(...own.splice(coverIndex, 1));
    return { moment, photos: own, sealed: sealedCounts.get(moment.id) ?? 0 };
  });

  const loose = photos
    .filter((photo) => !photo.momentId || !publicMomentIds.has(photo.momentId))
    .map((photo) => cardById.get(photo.id)!);

  const featuredRows = photos.filter((photo) => photo.featured);
  const featured = (featuredRows.length ? featuredRows : photos.slice(-5))
    .slice(0, 5)
    .map((photo) => cardById.get(photo.id)!);

  const stamps: Stamp[] = [];
  const seen = new Set<string>();
  const addStamp = (place: string | null, day: string | null, thumb: string | null) => {
    const name = place?.trim();
    if (!name || seen.has(name)) return;
    seen.add(name);
    stamps.push({ place: name, day, thumb });
  };
  for (const chapter of chapters) {
    addStamp(chapter.moment.place, chapter.moment.startsOn, chapter.photos[0]?.thumb ?? null);
  }
  for (const card of cards) addStamp(card.place, card.takenAt?.slice(0, 10) ?? null, card.thumb);
  stamps.sort((a, b) => (a.day ?? '9999').localeCompare(b.day ?? '9999'));

  const now = today();
  const since = isDay(settings.togetherSince) ? settings.togetherSince : now;
  const days = Math.max(1, dayNumber(since, now));

  return {
    settings,
    chapters,
    loose,
    featured,
    stamps,
    blessings,
    stats: {
      days,
      weekends: Math.floor(days / 7),
      places: stamps.length,
      photos: counts.total,
      hidden: counts.private,
      valentines: occurrences(since, now, '02-14'),
    },
  };
});
