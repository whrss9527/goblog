/** Stable pseudo-random number in [0, 1) from a string, so layouts don't jump between renders. */
export function seeded(key: string, salt = 0) {
  let h = 2166136261 ^ salt;
  for (let i = 0; i < key.length; i++) {
    h ^= key.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return ((h >>> 0) % 10000) / 10000;
}

export const tiltFor = (key: string, max = 4) => `${(seeded(key) * 2 - 1) * max}deg`;

const TAPES = [
  'rgba(236, 196, 186, 0.8)',
  'rgba(214, 226, 200, 0.8)',
  'rgba(246, 225, 170, 0.8)',
  'rgba(200, 214, 232, 0.8)',
  'rgba(232, 208, 226, 0.8)',
];

export const tapeFor = (key: string) => TAPES[Math.floor(seeded(key, 7) * TAPES.length)];

export const clampText = (value: string | null | undefined, max: number) =>
  (value ?? '').trim().slice(0, max);
