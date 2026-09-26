export type Visibility = 'public' | 'private';

/** 'a' signs in with ADMIN_EMAIL, 'b' with PARTNER_EMAIL. */
export type Partner = 'a' | 'b';

export type MomentKind =
  | 'meet'
  | 'date'
  | 'trip'
  | 'daily'
  | 'anniversary'
  | 'milestone';

export interface Photo {
  id: string;
  /** Storage key of the large (display) rendition. */
  lgKey: string;
  /** Storage key of the small (grid / polaroid) rendition. */
  smKey: string;
  /** Storage key of the untouched original, when the uploader kept it. */
  origKey: string | null;
  width: number;
  height: number;
  /** Tiny JPEG data URL shown while the real image loads. */
  blurData: string | null;
  /** Average colour, used for tape and placeholders. */
  color: string | null;
  /** Written on the front of the polaroid. */
  caption: string | null;
  /** Written on the back of the polaroid. */
  note: string | null;
  /** Wall-clock time the photo was taken, `YYYY-MM-DDTHH:mm:ss`. */
  takenAt: string | null;
  place: string | null;
  latitude: number | null;
  longitude: number | null;
  camera: string | null;
  author: Partner | null;
  momentId: string | null;
  visibility: Visibility;
  /** Shown in the hero, the envelope and the embed slideshow. */
  featured: boolean;
  /** Our own little heart, only visible to the two of us. */
  favorite: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Moment {
  id: string;
  title: string;
  kind: MomentKind;
  /** `YYYY-MM-DD` */
  startsOn: string;
  endsOn: string | null;
  place: string | null;
  story: string | null;
  coverPhotoId: string | null;
  visibility: Visibility;
  createdAt: string;
  updatedAt: string;
}

export type Attendance = 'yes' | 'no' | 'maybe';

export interface GuestNote {
  id: string;
  name: string;
  message: string | null;
  attending: Attendance | null;
  partySize: number | null;
  contact: string | null;
  approved: boolean;
  createdAt: string;
}

/**
 * What the browser gets for a photo. Deliberately leaves out GPS, camera,
 * the original file and anything else a guest does not need.
 */
export interface PhotoCard {
  id: string;
  src: string;
  thumb: string;
  width: number;
  height: number;
  blurData: string | null;
  color: string | null;
  caption: string | null;
  note: string | null;
  takenAt: string | null;
  place: string | null;
  author: string | null;
  isPrivate?: boolean;
  favorite?: boolean;
}

export interface Settings {
  siteTitle: string;
  partnerA: string;
  partnerB: string;
  /** Pressed into the wax seal, e.g. "J & M". */
  initials: string;
  /** `YYYY-MM-DD` — the day the red thread was tied. */
  togetherSince: string;
  /** `YYYY-MM-DD` — optional, the very first meeting. */
  firstMet: string;
  tagline: string;
  intro: string;
  envelopeEnabled: boolean;
  envelopeLine: string;
  weddingEnabled: boolean;
  /** `YYYY-MM-DD` */
  weddingDate: string;
  /** `HH:mm` */
  weddingTime: string;
  weddingVenue: string;
  weddingAddress: string;
  /** Optional "lng,lat" so the map buttons can drop an exact pin. */
  weddingLngLat: string;
  weddingInvitation: string;
  /** One line per item: "11:18 迎宾" */
  weddingSchedule: string;
  dressCode: string;
  rsvpEnabled: boolean;
  /** `YYYY-MM-DD` */
  rsvpDeadline: string;
  autoApproveNotes: boolean;
  /** An absolute URL, or a storage key uploaded from the settings page. */
  music: string;
  closingLine: string;
}
