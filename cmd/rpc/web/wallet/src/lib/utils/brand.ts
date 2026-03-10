const CANOPY_COLORS = [
  "#6EE7B7",
  "#38BDF8",
  "#C084FC",
  "#FBBF24",
  "#F472B6",
  "#60A5FA",
  "#F87171",
  "#34D399",
  "#A78BFA",
  "#F59E0B",
];

/**
 * Return a pseudo-random color from the palette based on a seed (string/number).
 * Falls back to a truly random pick if no seed provided.
 */
export function getCanopyAccent(seed?: string | number): string {
  if (seed === undefined || seed === null) {
    return CANOPY_COLORS[Math.floor(Math.random() * CANOPY_COLORS.length)];
  }

  const s = seed.toString();
  const hash = s.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return CANOPY_COLORS[hash % CANOPY_COLORS.length];
}

/**
 * Renders the canopy icon SVG with a configurable fill color.
 * The SVG scales to fill its container (uses 100% width/height).
 */
export function canopyIconSvg(color: string): string {
  return `<svg width="100%" height="100%" viewBox="0 0 441.8 441.79" xmlns="http://www.w3.org/2000/svg" fill="${color}"><path d="M101.33 40.32l-31.11 281.14c-1.28 11.53-6.44 22.28-14.64 30.49l-6.9 6.9C-21.26 271.82-15.83 144.26 64.9 63.53c9.85-9.82 20.4-18.54 31.46-26.14 2.25-1.54 5.27.22 4.97 2.93z"/><path d="M120.86 371.57l280.67-31.14c2.71-.3 4.47 2.72 2.93 4.97-7.6 11.09-16.33 21.64-26.19 31.5-80.65 80.65-208.02 86.12-295.01 16.44l7.13-7.13c8.2-8.2 18.95-13.36 30.47-14.64z"/><path d="M162.48 245.05l-45.66 45.66L147.66 13.25c.14-1.23.97-2.28 2.14-2.68C165.16 5.31 181 1.78 196.99.02c2.01-.22 3.7 1.51 3.48 3.51l-23.35 211.03c-1.28 11.53-6.44 22.28-14.64 30.49z"/><path d="M269.41 138.16l-45.66 45.66L243.84 2.97c.19-1.74 1.77-3.01 3.51-2.81 15.24 1.76 30.31 5.1 44.97 10.02 1.42.48 2.31 1.9 2.15 3.39l-10.41 94.11c-1.28 11.53-6.44 22.28-14.64 30.48z"/><path d="M406.49 106.54l-63.91 7.08c-8.3.92-15.32-6.1-14.39-14.4l7.11-63.9c.26-2.32 2.87-3.61 4.83-2.35 13.53 8.59 26.33 18.76 38.14 30.57 11.81 11.81 22.01 24.64 30.58 38.18 1.25 1.97-.04 4.57-2.36 4.83z"/><path d="M438.83 197.96l-180.24 20.01 45.59-45.59c8.2-8.2 18.95-13.36 30.48-14.64l93.58-10.37c1.49-.17 2.91.73 3.39 2.15 4.92 14.66 8.26 29.7 10.02 44.94.2 1.74-1.07 3.31-2.81 3.51z"/><path d="M441.78 244.81c-1.76 15.99-5.29 31.83-10.55 47.18-.4 1.17-1.45 2-2.68 2.14l-276.85 30.73 45.59-45.56c8.2-8.19 18.94-13.35 30.46-14.63l210.52-23.35c2.01-.22 3.73 1.47 3.51 3.48z"/></svg>`;
}

export const EXPLORER_NEON_GREEN = "#35cd48";
export const EXPLORER_NEON_BORDER = "#35cd48";
export const EXPLORER_ICON_GLOW =
  "text-[#35cd48] drop-shadow-[0_0_12px_rgba(53,205,72,0.4)]";
