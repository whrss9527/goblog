// Generates the soft illustrations in public/demo used by the sample album.
// Run with `node scripts/demo-art.mjs`; the output is committed, so this only
// needs to run again when a scene changes.
import fs from 'node:fs';
import path from 'node:path';

const OUT = path.join(path.dirname(new URL(import.meta.url).pathname), '..', 'public', 'demo');
fs.mkdirSync(OUT, { recursive: true });

let seed = 7;
const rand = () => ((seed = (seed * 16807) % 2147483647) / 2147483647);
const r = (a, b) => a + rand() * (b - a);

const grad = (id, stops, x2 = 0, y2 = 1) =>
  `<linearGradient id="${id}" x1="0" y1="0" x2="${x2}" y2="${y2}">${stops
    .map(([o, c]) => `<stop offset="${o}" stop-color="${c}"/>`)
    .join('')}</linearGradient>`;

const radial = (id, inner, outer) =>
  `<radialGradient id="${id}"><stop offset="0" stop-color="${inner}"/><stop offset="1" stop-color="${outer}" stop-opacity="0"/></radialGradient>`;

const finish = () => `
  <radialGradient id="vig" cx="50%" cy="50%" r="75%"><stop offset="0.6" stop-color="#000" stop-opacity="0"/><stop offset="1" stop-color="#2a1a10" stop-opacity="0.35"/></radialGradient>
  <filter id="grain"><feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" stitchTiles="stitch"/><feColorMatrix values="0 0 0 0 0.5 0 0 0 0 0.4 0 0 0 0 0.3 0 0 0 0.18 0"/></filter>`;

const overlay = (W, H) =>
  `<rect width="${W}" height="${H}" fill="url(#vig)"/><rect width="${W}" height="${H}" filter="url(#grain)" opacity="0.55"/>`;

/** Two little people holding hands, feet at (x, y). */
const couple = (x, y, s = 1, color = '#2b2230') => `
  <g transform="translate(${x} ${y}) scale(${s})" fill="${color}">
    <circle cx="-16" cy="-86" r="9"/><path d="M-26 -74h20l4 40h-7l-2 34h-6l-2-34h-4l-2 34h-6l-2-34h-7z"/>
    <circle cx="16" cy="-84" r="8.5"/><path d="M6 -73h20l10 44h-12l-1 29h-6l-2-29h-4l-2 29h-6l-1-29h-8z"/>
    <path d="M-6 -56 q6 6 12 0" stroke="${color}" stroke-width="4" fill="none"/>
  </g>`;

const stars = (W, H, n, top = 0.6) =>
  Array.from({ length: n }, () => `<circle cx="${r(0, W).toFixed(0)}" cy="${r(0, H * top).toFixed(0)}" r="${r(0.6, 2.2).toFixed(1)}" fill="#fff" opacity="${r(0.3, 1).toFixed(2)}"/>`).join('');

const heart = (x, y, s, color) =>
  `<path transform="translate(${x} ${y}) scale(${s})" d="M0 12S-14 3-14-5a7 7 0 0 1 14-3 7 7 0 0 1 14 3C14 3 0 12 0 12z" fill="${color}"/>`;

const waves = (W, y0, H, color) =>
  Array.from({ length: 9 }, (_, i) => {
    const y = y0 + (i + 1) * ((H - y0) / 10);
    const amp = 3 + i;
    let d = `M0 ${y}`;
    for (let x = 0; x <= W; x += 60) d += ` q15 ${-amp} 30 0 t30 0`;
    return `<path d="${d}" stroke="${color}" stroke-width="${1 + i * 0.25}" fill="none" opacity="${0.25 + i * 0.05}"/>`;
  }).join('');

const scenes = {
  lake: (W, H) => `
    <defs>${grad('sky', [[0, '#f6c9a6'], [0.55, '#f09a86'], [1, '#e9b7a8']])}${grad('water', [[0, '#e7a18f'], [1, '#8a6e8f']])}${radial('glow', '#fff3d6', '#f6b58a')}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <circle cx="${W * 0.62}" cy="${H * 0.5}" r="${H * 0.28}" fill="url(#glow)" opacity="0.9"/>
    <circle cx="${W * 0.62}" cy="${H * 0.52}" r="${H * 0.09}" fill="#fff1d8"/>
    <path d="M0 ${H * 0.56} C${W * 0.2} ${H * 0.5} ${W * 0.35} ${H * 0.53} ${W * 0.5} ${H * 0.55} S${W * 0.85} ${H * 0.5} ${W} ${H * 0.55} V${H * 0.6} H0z" fill="#9b6f86" opacity="0.7"/>
    <rect y="${H * 0.58}" width="${W}" height="${H * 0.42}" fill="url(#water)"/>
    <ellipse cx="${W * 0.62}" cy="${H * 0.7}" rx="${W * 0.05}" ry="${H * 0.12}" fill="#fff1d8" opacity="0.5"/>
    ${waves(W, H * 0.58, H, '#fff4e4')}
    <path d="M${W * 0.05} ${H} L${W * 0.05} ${H * 0.25}" stroke="#4a3740" stroke-width="10"/>
    ${Array.from({ length: 14 }, (_, i) => `<path d="M${W * 0.05} ${H * 0.28 + i * 6} q${40 + i * 6} ${40 + i * 12} ${60 + i * 8} ${160 + i * 18}" stroke="#4a3740" stroke-width="2.5" fill="none" opacity="0.8"/>`).join('')}
    <rect x="${W * 0.32}" y="${H * 0.78}" width="${W * 0.36}" height="${H * 0.03}" fill="#5a4048" opacity="0.8"/>
    ${couple(W * 0.5, H * 0.785, 1.1, '#3b2a33')}`,

  sea: (W, H) => `
    <defs>${grad('sky', [[0, '#fbd9b6'], [0.6, '#f3a58c'], [1, '#e98d7e']])}${grad('sea', [[0, '#7d8fb3'], [1, '#3f577f']])}${grad('sand', [[0, '#f2d3b0'], [1, '#e4b98f']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <circle cx="${W * 0.3}" cy="${H * 0.46}" r="${H * 0.08}" fill="#fff0d0"/>
    <rect y="${H * 0.5}" width="${W}" height="${H * 0.3}" fill="url(#sea)"/>
    ${waves(W, H * 0.5, H * 0.8, '#ffe3cc')}
    <path d="M0 ${H * 0.78} C${W * 0.3} ${H * 0.74} ${W * 0.6} ${H * 0.8} ${W} ${H * 0.75} V${H} H0z" fill="url(#sand)"/>
    <path d="M0 ${H * 0.785} C${W * 0.3} ${H * 0.745} ${W * 0.6} ${H * 0.805} ${W} ${H * 0.755}" stroke="#fff" stroke-width="4" fill="none" opacity="0.7"/>
    ${couple(W * 0.66, H * 0.9, 1.2, '#3a2a36')}
    ${heart(W * 0.3, H * 0.9, 1.4, '#c2343f')}`,

  lighthouse: (W, H) => `
    <defs>${grad('sky', [[0, '#40507e'], [0.6, '#9c86a8'], [1, '#f2b49a']])}${grad('sea', [[0, '#546b98'], [1, '#2c3b5e']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>${stars(W, H, 40, 0.35)}
    <rect y="${H * 0.66}" width="${W}" height="${H * 0.34}" fill="url(#sea)"/>${waves(W, H * 0.66, H, '#c8d4f0')}
    <path d="M${W * 0.55} ${H * 0.72} L${W * 1.05} ${H * 0.62} V${H * 0.8}z" fill="#20283f"/>
    <path d="M${W * 0.62} ${H * 0.66} l18 -${H * 0.36} h40 l18 ${H * 0.36}z" fill="#f4ede3"/>
    ${[0.36, 0.46, 0.56].map((f) => `<path d="M${W * 0.62 + 10} ${H * (0.3 + f * 0.6)} h${56}" stroke="#c2343f" stroke-width="18" opacity="0.9"/>`).join('')}
    <rect x="${W * 0.62 + 16}" y="${H * 0.26}" width="44" height="30" fill="#ffe9a6"/>
    <path d="M${W * 0.62 + 38} ${H * 0.28} L0 ${H * 0.12} L0 ${H * 0.36}z" fill="#fff1b8" opacity="0.28"/>
    <path d="M${W * 0.62 + 12} ${H * 0.26} l26 -22 26 22z" fill="#c2343f"/>`,

  shell: (W, H) => `
    <defs>${grad('sand', [[0, '#f4dcc0'], [1, '#e6c29d']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sand)"/>
    ${Array.from({ length: 160 }, () => `<circle cx="${r(0, W).toFixed(0)}" cy="${r(0, H).toFixed(0)}" r="${r(1, 3).toFixed(1)}" fill="#c9a07a" opacity="${r(0.2, 0.6).toFixed(2)}"/>`).join('')}
    <path d="M0 ${H * 0.2} C${W * 0.3} ${H * 0.28} ${W * 0.6} ${H * 0.12} ${W} ${H * 0.22} V0 H0z" fill="#9fc3d6" opacity="0.8"/>
    <path d="M0 ${H * 0.22} C${W * 0.3} ${H * 0.3} ${W * 0.6} ${H * 0.14} ${W} ${H * 0.24}" stroke="#fff" stroke-width="6" fill="none" opacity="0.8"/>
    <g transform="translate(${W * 0.5} ${H * 0.58}) scale(9)"><path d="M0 12S-14 3-14-5a7 7 0 0 1 14-3 7 7 0 0 1 14 3C14 3 0 12 0 12z" fill="#f6d5cf"/>
    ${Array.from({ length: 7 }, (_, i) => `<path d="M0 11 L${-12 + i * 4} -5" stroke="#dca79d" stroke-width="0.5"/>`).join('')}</g>
    <path d="M${W * 0.2} ${H * 0.85} q40 -30 80 0" stroke="#c2343f" stroke-width="4" fill="none" opacity="0.5"/>`,

  sleep: (W, H) => `
    <defs>${grad('bg', [[0, '#b7bfd0'], [1, '#8d97ad']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#bg)"/>
    <rect x="${W * 0.08}" y="${H * 0.12}" width="${W * 0.5}" height="${H * 0.42}" rx="30" fill="#dfe6f0"/>
    <rect x="${W * 0.1}" y="${H * 0.15}" width="${W * 0.46}" height="${H * 0.36}" rx="24" fill="#a6c1dc"/>
    <path d="M${W * 0.1} ${H * 0.4} q${W * 0.1} -40 ${W * 0.2} 0 t${W * 0.26} 0 V${H * 0.51} H${W * 0.1}z" fill="#7d9b7a"/>
    <path d="M0 ${H * 0.62} H${W} V${H} H0z" fill="#6b7489"/>
    <circle cx="${W * 0.62}" cy="${H * 0.6}" r="70" fill="#3b3342"/><path d="M${W * 0.52} ${H} q10 -${H * 0.28} ${W * 0.1} -${H * 0.3} q${W * 0.1} 0 ${W * 0.1} ${H * 0.3}z" fill="#4c4658"/>
    <circle cx="${W * 0.8}" cy="${H * 0.52}" r="72" fill="#2f2a36"/><path d="M${W * 0.68} ${H} q10 -${H * 0.36} ${W * 0.12} -${H * 0.38} q${W * 0.12} 0 ${W * 0.12} ${H * 0.38}z" fill="#3d3847"/>
    <text x="${W * 0.55}" y="${H * 0.4}" font-family="Georgia" font-size="46" fill="#fff" opacity="0.8">z z z</text>`,

  bookstore: (W, H) => `
    <defs>${grad('wall', [[0, '#e8c39a'], [1, '#c98a5a']])}${grad('win', [[0, '#8fa7bd'], [1, '#c3d0da']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#wall)"/>
    <rect x="${W * 0.1}" y="${H * 0.06}" width="${W * 0.8}" height="${H * 0.3}" rx="8" fill="url(#win)"/>
    ${Array.from({ length: 50 }, () => { const x = r(W * 0.1, W * 0.9), y = r(H * 0.06, H * 0.33); return `<path d="M${x.toFixed(0)} ${y.toFixed(0)} l-6 22" stroke="#fff" stroke-width="2" opacity="0.6"/>`; }).join('')}
    <path d="M${W * 0.5} ${H * 0.06} V${H * 0.36}" stroke="#7a5236" stroke-width="10"/>
    ${[0.44, 0.6, 0.76].map((f) => `<rect x="${W * 0.04}" y="${H * f}" width="${W * 0.92}" height="14" fill="#6d452b"/>
      ${Array.from({ length: 22 }, (_, i) => { const w = r(20, 36); const h = r(70, 120); return `<rect x="${(W * 0.06 + i * 36).toFixed(0)}" y="${(H * f - h).toFixed(0)}" width="${w.toFixed(0)}" height="${h.toFixed(0)}" fill="${['#c2343f', '#e8b04e', '#5b7f6e', '#3f5a86', '#e8d5b0', '#9a5a7a'][i % 6]}" opacity="0.9"/>`; }).join('')}`).join('')}
    <rect y="${H * 0.78}" width="${W}" height="${H * 0.22}" fill="#8a5a3a"/>
    ${couple(W * 0.5, H * 0.99, 2.1, '#3a2620')}`,

  cafe: (W, H) => `
    <defs>${grad('bg', [[0, '#e9cfb2'], [1, '#c79a73']])}${radial('lamp', '#fff3d6', '#e9cfb2')}</defs>
    <rect width="${W}" height="${H}" fill="url(#bg)"/>
    <circle cx="${W * 0.5}" cy="${H * 0.15}" r="${H * 0.4}" fill="url(#lamp)"/>
    <rect y="${H * 0.6}" width="${W}" height="${H * 0.4}" fill="#8c5d3b"/>
    ${[0.34, 0.66].map((f, i) => `
      <ellipse cx="${W * f}" cy="${H * 0.7}" rx="120" ry="30" fill="#f7efe4"/>
      <path d="M${W * f - 70} ${H * 0.52} h140 l-14 ${H * 0.18} h-112z" fill="#fbf6ee"/>
      <path d="M${W * f + 68} ${H * 0.56} q46 6 0 60" stroke="#fbf6ee" stroke-width="14" fill="none"/>
      <ellipse cx="${W * f}" cy="${H * 0.52}" rx="70" ry="16" fill="#9b6a45"/>
      ${heart(W * f, H * 0.515, 1.6, '#f3e2cc')}
      <path d="M${W * f - 16} ${H * 0.44} q-12 -26 6 -50 M${W * f + 14} ${H * 0.44} q-12 -26 6 -50" stroke="#fff" stroke-width="4" fill="none" opacity="${0.5 - i * 0.1}"/>`).join('')}`,

  kitchen: (W, H) => `
    <defs>${grad('bg', [[0, '#f1e3c8'], [1, '#dcc29a']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#bg)"/>
    ${Array.from({ length: 12 }, (_, i) => `<path d="M${i * W / 11} 0 V${H * 0.55}" stroke="#fff" stroke-width="3" opacity="0.5"/>`).join('')}
    <rect y="${H * 0.55}" width="${W}" height="${H * 0.45}" fill="#b9936a"/>
    <rect y="${H * 0.55}" width="${W}" height="16" fill="#8b6844"/>
    <path d="M${W * 0.3} ${H * 0.42} h${W * 0.4} l-20 ${H * 0.2} h-${W * 0.4 - 40}z" fill="#3a3436"/>
    <rect x="${W * 0.27}" y="${H * 0.4}" width="${W * 0.46}" height="24" rx="10" fill="#4a4446"/>
    <path d="M${W * 0.73} ${H * 0.41} h120" stroke="#3a3436" stroke-width="16" stroke-linecap="round"/>
    ${[0.4, 0.5, 0.6].map((f) => `<path d="M${W * f} ${H * 0.36} q-30 -40 0 -80 t0 -80" stroke="#6b6264" stroke-width="12" fill="none" opacity="0.55" stroke-linecap="round"/>`).join('')}
    <text x="${W * 0.08}" y="${H * 0.2}" font-family="Georgia" font-size="60" fill="#c2343f" opacity="0.8">oops!</text>`,

  cat: (W, H) => `
    <defs>${grad('win', [[0, '#f3d6a6'], [1, '#e9a88a']])}</defs>
    <rect width="${W}" height="${H}" fill="#8a7a6a"/>
    <rect x="${W * 0.1}" y="${H * 0.08}" width="${W * 0.8}" height="${H * 0.58}" fill="url(#win)"/>
    <path d="M${W * 0.5} ${H * 0.08} V${H * 0.66} M${W * 0.1} ${H * 0.37} H${W * 0.9}" stroke="#5b4c40" stroke-width="12"/>
    <rect y="${H * 0.66}" width="${W}" height="${H * 0.05}" fill="#5b4c40"/>
    <g fill="#2e2622" transform="translate(${W * 0.5} ${H * 0.66})">
      <ellipse cx="0" cy="-70" rx="110" ry="80"/><circle cx="-30" cy="-190" r="70"/>
      <path d="M-90 -230 l10 -70 40 45z M20 -230 l10 -70 -50 45z"/>
      <path d="M100 -40 q120 -10 90 -140" stroke="#2e2622" stroke-width="26" fill="none" stroke-linecap="round"/>
    </g>`,

  mountain: (W, H) => `
    <defs>${grad('sky', [[0, '#7fa6d6'], [1, '#dbe8f5']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <path d="M0 ${H * 0.8} L${W * 0.25} ${H * 0.3} L${W * 0.42} ${H * 0.5} L${W * 0.62} ${H * 0.18} L${W} ${H * 0.72} V${H} H0z" fill="#6a82a8"/>
    <path d="M${W * 0.62} ${H * 0.18} L${W * 0.55} ${H * 0.3} l30 10 20 -16 26 20 22 -8z M${W * 0.25} ${H * 0.3} L${W * 0.2} ${H * 0.38} l24 8 16 -10 18 8z" fill="#fff"/>
    <path d="M0 ${H * 0.86} Q${W * 0.3} ${H * 0.7} ${W * 0.6} ${H * 0.82} T${W} ${H * 0.78} V${H} H0z" fill="#e9f0f8"/>
    ${couple(W * 0.3, H * 0.9, 1.1, '#2f3d5c')}
    ${Array.from({ length: 30 }, () => `<circle cx="${r(0, W).toFixed(0)}" cy="${r(0, H).toFixed(0)}" r="${r(2, 5).toFixed(1)}" fill="#fff" opacity="0.8"/>`).join('')}`,

  stars: (W, H) => `
    <defs>${grad('sky', [[0, '#141a33'], [0.7, '#2c3553'], [1, '#6a5a7a']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <path d="M${-W * 0.1} ${H * 0.7} Q${W * 0.5} ${H * 0.1} ${W * 1.1} ${H * 0.05}" stroke="#c9c2ff" stroke-width="120" opacity="0.12" fill="none"/>
    ${stars(W, H, 220, 0.85)}
    <path d="M0 ${H * 0.82} L${W * 0.3} ${H * 0.62} L${W * 0.5} ${H * 0.74} L${W * 0.75} ${H * 0.58} L${W} ${H * 0.8} V${H} H0z" fill="#11142a"/>
    <path d="M${W * 0.62} ${H * 0.92} l60 -90 60 90z" fill="#e6a35a"/><path d="M${W * 0.62 + 60} ${H * 0.92} l-18 0 18 -40 18 40z" fill="#fff0c8"/>
    <path d="M0 ${H * 0.93} H${W}" stroke="#11142a" stroke-width="30"/>
    ${couple(W * 0.35, H * 0.94, 0.9, '#0b0d1c')}`,

  city: (W, H) => `
    <defs>${grad('sky', [[0, '#1d2140'], [1, '#6d5a86']])}${grad('river', [[0, '#3b3f63'], [1, '#1d2140']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>${stars(W, H, 50, 0.3)}
    ${Array.from({ length: 16 }, (_, i) => { const w = r(50, 90); const h = r(H * 0.15, H * 0.5); const x = i * (W / 15) - 20; return `<rect x="${x.toFixed(0)}" y="${(H * 0.62 - h).toFixed(0)}" width="${w.toFixed(0)}" height="${h.toFixed(0)}" fill="#2a2748"/>${Array.from({ length: 10 }, () => `<rect x="${(x + r(6, w - 12)).toFixed(0)}" y="${(H * 0.62 - r(10, h - 10)).toFixed(0)}" width="6" height="8" fill="#ffd98a" opacity="${r(0.4, 1).toFixed(2)}"/>`).join('')}`; }).join('')}
    <path d="M${W * 0.7} ${H * 0.62} V${H * 0.12} M${W * 0.7} ${H * 0.22} m-26 0 a26 26 0 1 0 52 0 a26 26 0 1 0 -52 0 M${W * 0.7} ${H * 0.38} m-18 0 a18 18 0 1 0 36 0 a18 18 0 1 0 -36 0" stroke="#e36a8c" stroke-width="12" fill="#2a2748"/>
    <rect y="${H * 0.62}" width="${W}" height="${H * 0.38}" fill="url(#river)"/>
    ${Array.from({ length: 40 }, () => `<rect x="${r(0, W).toFixed(0)}" y="${r(H * 0.64, H * 0.9).toFixed(0)}" width="${r(10, 40).toFixed(0)}" height="3" fill="#ffd98a" opacity="${r(0.2, 0.6).toFixed(2)}"/>`).join('')}
    <rect y="${H * 0.9}" width="${W}" height="${H * 0.1}" fill="#15172e"/>
    ${couple(W * 0.25, H * 0.95, 1, '#0c0d1c')}`,

  balcony: (W, H) => `
    <defs>${grad('sky', [[0, '#2a2544'], [1, '#c9776a']])}${radial('glow', '#ffdc9a', '#c9776a')}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>${stars(W, H, 60, 0.4)}
    <circle cx="${W * 0.5}" cy="${H * 0.68}" r="${H * 0.3}" fill="url(#glow)" opacity="0.8"/>
    <rect x="${W * 0.2}" y="${H * 0.7}" width="${W * 0.6}" height="16" fill="#5a3a36"/>
    <path d="M${W * 0.3} ${H * 0.72} V${H} M${W * 0.7} ${H * 0.72} V${H}" stroke="#5a3a36" stroke-width="12"/>
    ${[0.42, 0.58].map((f) => `<rect x="${W * f - 9}" y="${H * 0.58}" width="18" height="${H * 0.12}" fill="#fff4e0"/><path d="M${W * f} ${H * 0.58} q-10 -20 0 -40 q10 20 0 40z" fill="#ffc85a"/>`).join('')}
    ${heart(W * 0.5, H * 0.66, 2, '#c2343f')}
    ${Array.from({ length: 18 }, (_, i) => `<path d="M${i * W / 17} ${H * 0.86} V${H}" stroke="#3a2a30" stroke-width="6"/>`).join('')}
    <path d="M0 ${H * 0.86} H${W}" stroke="#3a2a30" stroke-width="10"/>`,

  snownight: (W, H) => `
    <defs>${grad('sky', [[0, '#1b2440'], [1, '#3a4a6e']])}${radial('lamp', '#ffe6a8', '#3a4a6e')}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    ${[0.18, 0.82].map((f) => `<rect x="${W * f - 60}" y="${H * 0.2}" width="120" height="${H * 0.6}" fill="#2a3150"/>${Array.from({ length: 8 }, (_, i) => `<rect x="${W * f - 40 + (i % 2) * 50}" y="${H * 0.25 + Math.floor(i / 2) * 110}" width="30" height="50" fill="#ffd98a" opacity="0.8"/>`).join('')}`).join('')}
    <circle cx="${W * 0.5}" cy="${H * 0.36}" r="${H * 0.18}" fill="url(#lamp)" opacity="0.9"/>
    <path d="M${W * 0.5} ${H * 0.4} V${H * 0.82}" stroke="#141a30" stroke-width="10"/><rect x="${W * 0.5 - 18}" y="${H * 0.33}" width="36" height="46" rx="6" fill="#ffe6a8"/>
    <path d="M0 ${H * 0.8} Q${W * 0.5} ${H * 0.76} ${W} ${H * 0.8} V${H} H0z" fill="#e8eef8"/>
    <g transform="translate(${W * 0.5} ${H * 0.86})" fill="#141a30">
      <circle cx="-60" cy="-150" r="18"/><path d="M-82 -130h44l8 90h-16l-4 40h-24l-4-40h-12z"/>
      <circle cx="40" cy="-110" r="17"/><path d="M20 -92h40l10 20-10 12v40h-40v-40l-10-12z"/>
    </g>
    ${heart(W * 0.5, H * 0.6, 1.6, '#e8505b')}
    ${Array.from({ length: 140 }, () => `<circle cx="${r(0, W).toFixed(0)}" cy="${r(0, H).toFixed(0)}" r="${r(1.5, 5).toFixed(1)}" fill="#fff" opacity="${r(0.4, 0.95).toFixed(2)}"/>`).join('')}`,

  ring: (W, H) => `
    <defs>${grad('bg', [[0, '#f3dcc0'], [1, '#d9b77e']])}${radial('shine', '#fffbe8', '#f3dcc0')}</defs>
    <rect width="${W}" height="${H}" fill="url(#bg)"/>
    <circle cx="${W * 0.5}" cy="${H * 0.42}" r="${H * 0.4}" fill="url(#shine)"/>
    <path d="M${W * 0.3} ${H * 0.62} h${W * 0.4} v${H * 0.26} h-${W * 0.4}z" fill="#9e2430"/>
    <path d="M${W * 0.3} ${H * 0.62} l30 -${H * 0.28} h${W * 0.4 - 60} l30 ${H * 0.28}z" fill="#b8323e"/>
    <rect x="${W * 0.36}" y="${H * 0.6}" width="${W * 0.28}" height="${H * 0.08}" rx="10" fill="#f3e4d0"/>
    <circle cx="${W * 0.5}" cy="${H * 0.52}" r="${H * 0.1}" fill="none" stroke="#d8a94c" stroke-width="18"/>
    <path d="M${W * 0.5} ${H * 0.34} l-26 20 26 22 26 -22z" fill="#eaf6ff" stroke="#b9d4e6" stroke-width="3"/>
    ${Array.from({ length: 6 }, (_, i) => `<path d="M${W * 0.5 + Math.cos(i) * 110} ${H * 0.3 + Math.sin(i) * 70} l0 -20 m-10 10 h20" stroke="#fff" stroke-width="4"/>`).join('')}`,

  flowers: (W, H) => `
    <defs>${grad('sky', [[0, '#f9e7dc'], [1, '#f3c9c9']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <path d="M0 ${H * 0.55} Q${W * 0.5} ${H * 0.45} ${W} ${H * 0.55} V${H} H0z" fill="#9dbb8c"/>
    ${Array.from({ length: 70 }, () => { const x = r(0, W), y = r(H * 0.55, H), s = r(0.6, 1.6); const c = ['#e98a95', '#f7d6d0', '#f3cf9e', '#fff', '#d9505c'][Math.floor(r(0, 5))]; return `<g transform="translate(${x.toFixed(0)} ${y.toFixed(0)}) scale(${s.toFixed(2)})"><path d="M0 0 v24" stroke="#5e8a58" stroke-width="3"/>${[0, 72, 144, 216, 288].map((a) => `<ellipse rx="6" ry="11" transform="rotate(${a}) translate(0 -10)" fill="${c}"/>`).join('')}<circle r="5" fill="#f2b84b"/></g>`; }).join('')}`,

  umbrella: (W, H) => `
    <defs>${grad('bg', [[0, '#a8b6c7'], [1, '#6f7f94']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#bg)"/>
    ${Array.from({ length: 160 }, () => { const x = r(0, W), y = r(0, H); return `<path d="M${x.toFixed(0)} ${y.toFixed(0)} l-8 30" stroke="#e8eef6" stroke-width="2" opacity="${r(0.3, 0.8).toFixed(2)}"/>`; }).join('')}
    <path d="M0 ${H * 0.82} H${W} V${H} H0z" fill="#56647a"/>
    ${Array.from({ length: 10 }, () => `<ellipse cx="${r(0, W).toFixed(0)}" cy="${r(H * 0.84, H).toFixed(0)}" rx="${r(20, 60).toFixed(0)}" ry="6" fill="#8fa0b8" opacity="0.6"/>`).join('')}
    <path d="M${W * 0.2} ${H * 0.42} Q${W * 0.5} ${H * 0.12} ${W * 0.8} ${H * 0.42} q-${W * 0.05} -30 -${W * 0.1} 0 q-${W * 0.05} -30 -${W * 0.1} 0 q-${W * 0.05} -30 -${W * 0.1} 0 q-${W * 0.05} -30 -${W * 0.1} 0 q-${W * 0.05} -30 -${W * 0.1} 0 q-${W * 0.05} -30 -${W * 0.1} 0z" fill="#c2343f"/>
    <path d="M${W * 0.5} ${H * 0.2} V${H * 0.62}" stroke="#2b2230" stroke-width="6"/>
    ${couple(W * 0.5, H * 0.86, 1.9, '#2b2230')}`,

  picnic: (W, H) => `
    <defs>${grad('sky', [[0, '#cfe6f2'], [1, '#f6f0d8']])}</defs>
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    <circle cx="${W * 0.82}" cy="${H * 0.18}" r="${H * 0.08}" fill="#fff4c2"/>
    <path d="M0 ${H * 0.5} Q${W * 0.5} ${H * 0.38} ${W} ${H * 0.5} V${H} H0z" fill="#9bbf8a"/>
    <path d="M${W * 0.12} ${H * 0.2} q40 -80 90 0 q60 -40 70 30 q40 30 -20 60 h-150 q-50 -40 10 -90z" fill="#6f9a62"/><path d="M${W * 0.12 + 70} ${H * 0.44} V${H * 0.55}" stroke="#6b4b36" stroke-width="18"/>
    <path d="M${W * 0.25} ${H * 0.7} L${W * 0.75} ${H * 0.62} L${W * 0.85} ${H * 0.9} L${W * 0.3} ${H * 0.98}z" fill="#f1efe8"/>
    ${Array.from({ length: 6 }, (_, i) => `<path d="M${W * (0.26 + i * 0.1)} ${H * (0.7 - i * 0.015)} L${W * (0.31 + i * 0.1)} ${H * (0.98 - i * 0.015)}" stroke="#c2343f" stroke-width="12" opacity="0.8"/>`).join('')}
    <rect x="${W * 0.45}" y="${H * 0.7}" width="90" height="60" rx="8" fill="#b88a5a"/><path d="M${W * 0.45 + 10} ${H * 0.7} q35 -40 70 0" stroke="#8a6440" stroke-width="6" fill="none"/>
    ${heart(W * 0.62, H * 0.8, 1.5, '#e98a95')}`,
};

const list = [
  ['lake'], ['sea'], ['lighthouse', true], ['shell'], ['sleep'], ['bookstore', true], ['cafe'], ['kitchen'],
  ['cat', true], ['mountain'], ['stars'], ['city'], ['balcony'], ['snownight', true], ['ring'], ['flowers'],
  ['umbrella', true], ['picnic'],
];

for (const [name, portrait] of list) {
  const W = portrait ? 900 : 1200;
  const H = portrait ? 1200 : 900;
  seed = [...name].reduce((a, c) => a * 31 + c.charCodeAt(0), 7) % 2147483646 || 7;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${W} ${H}" width="${W}" height="${H}"><defs>${finish()}</defs>${scenes[name](W, H)}${overlay(W, H)}</svg>\n`;
  fs.writeFileSync(path.join(OUT, `${name}.svg`), svg.replace(/\n\s+/g, '\n'));
}
console.log(`wrote ${list.length} illustrations to ${OUT}`);
