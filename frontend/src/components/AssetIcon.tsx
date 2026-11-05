"use client";

import { useId } from "react";

export type AssetIconProps = {
  symbol?: string | null;
  size?: number;
  className?: string;
};

const NAVY = "#16324F";
const GOLD = "#C9A24B";
const SERIF = "Georgia, 'Times New Roman', serif";

function normalize(symbol: string): string {
  const s = symbol.trim().toLowerCase();
  if (["eth", "weth", "steth", "reth"].includes(s)) return "eth";
  if (s === "usdc") return "usdc";
  if (s === "usdt") return "usdt";
  if (["cnpy", "canopy"].includes(s)) return "canopy";
  if (["btc", "bitcoin", "wbtc", "tbtc"].includes(s)) return "bitcoin";
  if (s === "nusd") return "nusd";
  if (s === "arbor") return "arbor";
  return s;
}

function Glyph({ k }: { k: string }) {
  switch (k) {
    case "arbor":
      return (
        <>
          <g stroke={NAVY} strokeWidth={12} strokeLinecap="round" fill="none">
            <line x1={0} y1={88} x2={0} y2={18} />
            <line x1={0} y1={18} x2={-52} y2={-32} />
            <line x1={0} y1={18} x2={52} y2={-32} />
            <line x1={-52} y1={-32} x2={-70} y2={-82} />
            <line x1={-52} y1={-32} x2={-42} y2={-92} />
            <line x1={52} y1={-32} x2={70} y2={-82} />
            <line x1={52} y1={-32} x2={42} y2={-92} />
            <line x1={0} y1={18} x2={0} y2={-78} />
          </g>
          <g fill={NAVY}>
            <circle cx={0} cy={18} r={9} />
            <circle cx={-52} cy={-32} r={7.5} />
            <circle cx={52} cy={-32} r={7.5} />
          </g>
          <g fill={GOLD}>
            <circle cx={-70} cy={-82} r={7} />
            <circle cx={-42} cy={-92} r={7} />
            <circle cx={0} cy={-78} r={7} />
            <circle cx={42} cy={-92} r={7} />
            <circle cx={70} cy={-82} r={7} />
          </g>
          <path d="M -84 -70 Q 0 -140 84 -70" fill="none" stroke={NAVY} strokeWidth={3} opacity={0.45} />
          <line x1={-44} y1={88} x2={44} y2={88} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "nusd":
      return (
        <>
          <g stroke={NAVY} strokeWidth={19} strokeLinecap="round" strokeLinejoin="round" fill="none">
            <line x1={-54} y1={60} x2={-54} y2={-60} />
            <line x1={54} y1={60} x2={54} y2={-60} />
            <line x1={-54} y1={-60} x2={54} y2={60} />
          </g>
          <circle cx={0} cy={0} r={9} fill={GOLD} stroke="#F7F3EA" strokeWidth={3} />
          <line x1={0} y1={82} x2={0} y2={102} stroke={NAVY} strokeWidth={5} strokeLinecap="round" />
          <circle cx={0} cy={110} r={6} fill={GOLD} />
          <line x1={-44} y1={132} x2={44} y2={132} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "canopy":
      return (
        <>
          <path d="M 0 -78 C 62 -58 62 58 0 88 C -62 58 -62 -58 0 -78 Z" fill="none" stroke={NAVY} strokeWidth={12} strokeLinejoin="round" />
          <line x1={0} y1={-58} x2={0} y2={72} stroke={GOLD} strokeWidth={6} strokeLinecap="round" />
          <g stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.6}>
            <line x1={0} y1={-20} x2={-30} y2={-46} />
            <line x1={0} y1={-20} x2={30} y2={-46} />
            <line x1={0} y1={26} x2={-32} y2={4} />
            <line x1={0} y1={26} x2={32} y2={4} />
          </g>
          <line x1={-44} y1={118} x2={44} y2={118} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "eth":
      return (
        <>
          <polygon points="0,-92 62,10 0,48 -62,10" fill="none" stroke={NAVY} strokeWidth={10} strokeLinejoin="round" />
          <polygon points="0,64 62,18 0,96 -62,18" fill={NAVY} opacity={0.18} stroke={NAVY} strokeWidth={10} strokeLinejoin="round" />
          <line x1={-62} y1={10} x2={62} y2={10} stroke={GOLD} strokeWidth={4} opacity={0.8} />
          <line x1={-44} y1={128} x2={44} y2={128} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "bitcoin":
      return (
        <>
          <text x={0} y={38} textAnchor="middle" fontSize={128} fontWeight="bold" fill={NAVY} fontFamily={SERIF}>B</text>
          <g stroke={GOLD} strokeWidth={7} strokeLinecap="round">
            <line x1={-14} y1={-92} x2={-14} y2={-68} />
            <line x1={16} y1={-92} x2={16} y2={-68} />
            <line x1={-14} y1={68} x2={-14} y2={92} />
            <line x1={16} y1={68} x2={16} y2={92} />
          </g>
          <line x1={-44} y1={128} x2={44} y2={128} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "usdc":
      return (
        <>
          <circle r={76} fill="none" stroke={NAVY} strokeWidth={9} />
          <text x={0} y={34} textAnchor="middle" fontSize={92} fontWeight="bold" fill={NAVY} fontFamily={SERIF}>$</text>
          <line x1={-44} y1={128} x2={44} y2={128} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    case "usdt":
      return (
        <>
          <g stroke={NAVY} strokeWidth={14} strokeLinecap="round">
            <line x1={0} y1={-70} x2={0} y2={70} />
            <line x1={-56} y1={-70} x2={56} y2={-70} />
            <line x1={-40} y1={-24} x2={40} y2={-24} />
          </g>
          <circle cx={0} cy={70} r={8} fill={GOLD} />
          <line x1={-44} y1={128} x2={44} y2={128} stroke={NAVY} strokeWidth={5} strokeLinecap="round" opacity={0.75} />
        </>
      );
    default:
      return (
        <text x={0} y={52} textAnchor="middle" fontSize={150} fontWeight="bold" fill={NAVY} fontFamily={SERIF}>
          {k.charAt(0).toUpperCase()}
        </text>
      );
  }
}

export function AssetIcon({ symbol, size = 32, className }: AssetIconProps) {
  const uid = useId();
  const gradId = `parchment-${uid}`;
  const k = normalize(symbol ?? "");
  return (
    <svg
      viewBox="-186 -186 372 372"
      width={size}
      height={size}
      className={className}
      role="img"
      aria-label={symbol || "asset"}
    >
      <defs>
        <linearGradient id={gradId} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#F7F3EA" />
          <stop offset="100%" stopColor="#EDE6D6" />
        </linearGradient>
      </defs>
      <circle r={180} fill={`url(#${gradId})`} stroke={NAVY} strokeWidth={6} />
      <circle r={161} fill="none" stroke={GOLD} strokeWidth={1.5} opacity={0.55} />
      <circle r={128} fill="none" stroke="#8FA0B3" strokeWidth={2.5} opacity={0.4} />
      <Glyph k={k} />
    </svg>
  );
}
