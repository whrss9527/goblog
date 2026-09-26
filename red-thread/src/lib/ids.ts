import { customAlphabet } from 'nanoid';

const alphabet = '0123456789abcdefghijklmnopqrstuvwxyz';

/** Short, URL-friendly ids for rows. */
export const newId = customAlphabet(alphabet, 12);

/** Long random token that makes storage keys unguessable. */
export const newToken = customAlphabet(alphabet, 20);

/**
 * Every photo file lives at `photos/<id>-<token>-<variant>.<ext>`: the token
 * keeps private photos out of reach even on buckets with a public domain.
 */
export const PHOTO_KEY = /^photos\/[a-z0-9]{12}-[a-z0-9]{20}-(lg|sm|orig)\.(jpg|jpeg|png|webp|heic|heif|gif|avif)$/;
export const MEDIA_KEY = /^media\/[a-z0-9]{20}\.(mp3|m4a|aac|ogg|wav)$/;
/** Illustrations shipped in /public/demo for the sample album. */
export const DEMO_KEY = /^demo\/[a-z0-9-]+\.svg$/;

export const isStorageKey = (key: string) => PHOTO_KEY.test(key) || MEDIA_KEY.test(key);
