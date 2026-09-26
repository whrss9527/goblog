import 'server-only';
import { query } from './db';
import { insertMoment, type MomentInput } from './moments';
import { insertPhoto } from './photos';
import { getSettings, saveSettings } from './settings';
import { addDays, today } from './dates';
import type { Partner, Visibility } from './types';

/**
 * A sample album built from the illustrations in /public/demo, so a fresh
 * install shows what the pages look like before the first real upload.
 * Everything it creates has an id starting with `demo-` and can be removed
 * with one click.
 */

type DemoPhoto = {
  art: string;
  portrait?: boolean;
  caption: string;
  note?: string;
  takenAt: string;
  place?: string;
  author?: Partner;
  visibility?: Visibility;
  featured?: boolean;
  favorite?: boolean;
  color: string;
};

type DemoMoment = MomentInput & { key: string; photos: DemoPhoto[] };

const MOMENTS: DemoMoment[] = [
  {
    key: 'meet',
    title: '第一次见面',
    kind: 'meet',
    startsOn: '2020-04-11',
    endsOn: null,
    place: '杭州 · 西湖边的书店',
    story: '那天下着小雨，你站在书架前翻一本《小王子》。\n我假装找书，在你身后绕了三圈，才鼓起勇气问你：这本好看吗？',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'bookstore', portrait: true, caption: '你偷偷看我的那一眼', note: '其实我也偷偷看了你很多眼。', takenAt: '2020-04-11T15:20:00', author: 'a', featured: true, color: '#c98a5a' },
      { art: 'cafe', caption: '两杯拿铁，一个下午', takenAt: '2020-04-11T16:40:00', author: 'b', color: '#b98763' },
    ],
  },
  {
    key: 'together',
    title: '在一起啦',
    kind: 'milestone',
    startsOn: '2020-05-20',
    endsOn: null,
    place: '杭州 · 白堤',
    story: '你说“那就试试看吧”。\n那天的晚风很软，湖面把整个夕阳都装了进去。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'lake', caption: '5 月 20 日，晴', note: '从这天起，“我”变成了“我们”。', takenAt: '2020-05-20T18:42:00', author: 'b', featured: true, favorite: true, color: '#e39a78' },
    ],
  },
  {
    key: 'sea',
    title: '第一次一起看海',
    kind: 'trip',
    startsOn: '2020-10-02',
    endsOn: '2020-10-05',
    place: '厦门 · 鼓浪屿',
    story: '你说想看海，我们就订了第二天的车票。\n海风很大，你的头发一直吹到我脸上，我一点都没躲。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'sea', caption: '海是蓝的，你是甜的', takenAt: '2020-10-03T17:55:00', author: 'a', featured: true, color: '#e7a07f' },
      { art: 'lighthouse', portrait: true, caption: '灯塔下面等日落', note: '那天你许的愿，到现在也没告诉我。', takenAt: '2020-10-03T18:30:00', author: 'b', color: '#6d7fa8' },
      { art: 'shell', caption: '捡到一颗心形的贝壳', takenAt: '2020-10-04T10:12:00', author: 'b', color: '#e8cfb5' },
      { art: 'sleep', caption: '回程的车上', note: '你靠着我睡了一路，我的肩膀麻了四个小时。', takenAt: '2020-10-05T15:05:00', author: 'a', visibility: 'private', color: '#9aa3b5' },
    ],
  },
  {
    key: 'porridge',
    title: '煮糊的第一锅粥',
    kind: 'daily',
    startsOn: '2021-01-17',
    endsOn: null,
    place: '我们的小家',
    story: '说好周末给你做早饭，结果把锅底烧黑了。\n你笑了整整一个早上，然后说：没关系，以后的早饭我们一起做。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'kitchen', caption: '厨房事故现场', takenAt: '2021-01-17T08:30:00', author: 'b', color: '#d6b48a' },
      { art: 'cat', portrait: true, caption: '唯一的目击证人', takenAt: '2021-01-17T09:10:00', author: 'a', color: '#8a7a6a' },
    ],
  },
  {
    key: 'snow',
    title: '雪山下许的愿',
    kind: 'trip',
    startsOn: '2022-01-22',
    endsOn: '2022-01-26',
    place: '丽江 · 玉龙雪山',
    story: '海拔四千多米，你喘着气还要比剪刀手。\n晚上的星星多得像撒出来的糖。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'mountain', caption: '离天空很近的地方', takenAt: '2022-01-23T11:00:00', author: 'a', color: '#7f9cc4' },
      { art: 'stars', caption: '数不清的星星', note: '我在心里许了一个愿，和你有关。', takenAt: '2022-01-24T22:40:00', author: 'b', featured: true, color: '#2c3553' },
    ],
  },
  {
    key: 'day1000',
    title: '在一起的第 1000 天',
    kind: 'anniversary',
    startsOn: '2023-02-13',
    endsOn: null,
    place: '上海',
    story: '我们回到第一次约会的那家面馆，老板居然还记得我们。\n一千天，好像只是一眨眼。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'city', caption: '外滩的风', takenAt: '2023-02-13T20:15:00', author: 'b', color: '#3b3f63' },
    ],
  },
  {
    key: 'secret',
    title: '只属于我们的小秘密',
    kind: 'date',
    startsOn: '2024-08-10',
    endsOn: null,
    place: '家里的阳台',
    story: '这一页不给别人看。',
    coverPhotoId: null,
    visibility: 'private',
    photos: [
      { art: 'balcony', caption: '阳台上的烛光晚餐', takenAt: '2024-08-10T20:00:00', author: 'a', visibility: 'private', favorite: true, color: '#c9776a' },
    ],
  },
  {
    key: 'yes',
    title: '我说，我愿意',
    kind: 'milestone',
    startsOn: '2025-12-24',
    endsOn: null,
    place: '哈尔滨 · 中央大街',
    story: '零下二十度，你单膝跪在雪地里，手抖得差点把戒指掉进雪里。\n我哭着笑着说了“我愿意”。',
    coverPhotoId: null,
    visibility: 'public',
    photos: [
      { art: 'snownight', portrait: true, caption: '零下二十度的“我愿意”', note: '以后的每一个冬天，都要和你一起过。', takenAt: '2025-12-24T21:05:00', author: 'b', featured: true, favorite: true, color: '#3a4a6e' },
      { art: 'ring', caption: '手抖的那个人', takenAt: '2025-12-24T21:08:00', author: 'a', color: '#d9b77e' },
    ],
  },
];

const LOOSE: DemoPhoto[] = [
  { art: 'flowers', caption: '路边的小花，送给你', takenAt: '2021-04-03T10:20:00', place: '杭州', author: 'a', color: '#e6a3b0' },
  { art: 'umbrella', portrait: true, caption: '一把伞，两个人', takenAt: '2022-06-18T19:00:00', place: '上海', author: 'b', color: '#7d8fa3' },
];

export async function loadDemo() {
  await clearDemo();
  let n = 0;
  const add = async (photo: DemoPhoto, momentId: string | null, momentPlace: string | null) => {
    n += 1;
    const id = `demo-p${String(n).padStart(2, '0')}`;
    const key = `demo/${photo.art}.svg`;
    await insertPhoto({
      id,
      lgKey: key,
      smKey: key,
      origKey: null,
      width: photo.portrait ? 900 : 1200,
      height: photo.portrait ? 1200 : 900,
      blurData: null,
      color: photo.color,
      caption: photo.caption,
      note: photo.note ?? null,
      takenAt: photo.takenAt,
      place: photo.place ?? momentPlace,
      latitude: null,
      longitude: null,
      camera: null,
      author: photo.author ?? null,
      momentId,
      visibility: photo.visibility ?? 'public',
      featured: photo.featured ?? false,
      favorite: photo.favorite ?? false,
    });
  };
  for (const moment of MOMENTS) {
    const { key, photos, ...input } = moment;
    const id = `demo-${key}`;
    await insertMoment(id, input);
    for (const photo of photos) await add(photo, id, input.place);
  }
  for (const photo of LOOSE) await add(photo, null, null);

  // "那年今日" should have something to show on the day you try the demo.
  const now = today();
  await add(
    { art: 'picnic', caption: '去年今天的野餐', note: '明年今天，我们也要一起。', takenAt: `${addDays(now, -365).slice(0, 10)}T14:00:00`, author: 'b', visibility: 'private', color: '#9bbf8a' },
    null,
    '公园的草坪',
  );

  const settings = await getSettings();
  if (!settings.weddingDate) {
    const year = Number(now.slice(0, 4)) + 1;
    await saveSettings({
      weddingEnabled: true,
      weddingDate: `${year}-05-20`,
      weddingTime: '11:58',
      weddingVenue: '湖畔的草坪婚礼',
      weddingAddress: '杭州市西湖区北山街 1 号（示例地址）',
      dressCode: '浅色系，穿得舒服就好',
    });
  }
}

export async function clearDemo() {
  await query(`DELETE FROM photos WHERE id LIKE 'demo-%'`);
  await query(`DELETE FROM moments WHERE id LIKE 'demo-%'`);
}

export async function hasDemo() {
  const rows = await query<{ n: number }>(`SELECT count(*)::int AS n FROM photos WHERE id LIKE 'demo-%'`);
  return Number(rows[0]?.n ?? 0) > 0;
}

