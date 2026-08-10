"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useWalletStore } from "@/lib/wallet";
import { WalletConnect } from "./WalletConnect";
import { useRoles, type Roles } from "@/lib/hooks/useRoles";

const NAV = [
  { href: "/", label: "Markets" },
  { href: "/nusd", label: "NUSD" },
  { href: "/send", label: "Send" },
  { href: "/oracle", label: "Oracle" },
  { href: "/liquidate", label: "Liquidation" }, { href: "/monitor", label: "Monitor" },
  { href: "/events", label: "Events" },
  { href: "/quests", label: "Quests" },
  { href: "/governance", label: "Governance" }, { href: "/admin", label: "Authority" },
  { href: "/tx", label: "Transactions" },
];

type ChainStatus = {
  height: number | null;
  blockTime: number | null;
  online: boolean;
};

function shortAddress(address: string): string {
  if (!address) return "";
  if (address.length <= 12) return address;
  return `${address.slice(0, 6)}…${address.slice(-4)}`;
}

function StatusChip() {
  const [status, setStatus] = useState<ChainStatus>({
    height: null,
    blockTime: null,
    online: false,
  });

  const last = useRef<{ h: number; t: number } | null>(null);

  useEffect(() => {
    let alive = true;

    async function poll() {
      try {
        const res = await fetch("/canopy-rpc/v1/query/height", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}",
        });

        if (!res.ok) throw new Error("bad status");

        const data = (await res.json()) as { height?: number };
        const h = typeof data.height === "number" ? data.height : null;

        if (!alive || h === null) return;

        const now = Date.now();
        let bt: number | null = null;

        if (last.current && h > last.current.h) {
          const dt = (now - last.current.t) / 1000;
          const dh = h - last.current.h;
          bt = dh > 0 ? Math.max(1, Math.round(dt / dh)) : null;
        }

        last.current = { h, t: now };

        setStatus({ height: h, blockTime: bt, online: true });
      } catch {
        if (alive) {
          setStatus((s) => ({ ...s, online: false }));
        }
      }
    }

    poll();
    const id = setInterval(poll, 4000);

    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  return (
    <div className="inline-flex items-center gap-2 rounded-full glass px-3 py-1.5 text-xs backdrop-blur">
      <span className="relative flex h-2 w-2">
        {status.online && (
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-70" />
        )}
        <span
          className={`relative inline-flex h-2 w-2 rounded-full ${
            status.online ? "bg-emerald-400" : "bg-rose-400"
          }`}
        />
      </span>

      {status.online ? (
        <span className="flex items-center gap-1.5 text-zinc-400">
          <span className="text-zinc-300">Canopy mainnet</span>
          <span className="text-zinc-600">·</span>
          <span>
            block{" "}
            <span className="tabular-nums text-zinc-200">
              {status.height?.toLocaleString() ?? "--"}
            </span>
          </span>
          {status.blockTime !== null && (
            <>
              <span className="text-zinc-600">·</span>
              <span className="tabular-nums">~{status.blockTime}s</span>
            </>
          )}
        </span>
      ) : (
        <span className="text-rose-300/90">Canopy offline</span>
      )}
    </div>
  );
}

function WalletChip() {
  const wallet = useWalletStore();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (wallet.isConnected) setOpen(false);
  }, [wallet.isConnected]);

  if (wallet.isConnected) {
    return (
      <div className="inline-flex items-center gap-2 rounded-full glass px-3 py-1.5 text-xs backdrop-blur">
        <span className="grid h-5 w-5 place-items-center rounded-full bg-gradient-to-br from-indigo-400 to-emerald-400 text-[9px] font-bold text-[#05070d]">
          ◆
        </span>
        <span className="tabular-nums text-zinc-200">
          {shortAddress(wallet.address ?? "")}
        </span>
      </div>
    );
  }

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center gap-2 rounded-full btn-brand px-4 py-1.5 text-xs font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400"
      >
        Connect wallet
      </button>

      {open && (
        <>
          <button
            type="button"
            aria-label="Close connect panel"
            onClick={() => setOpen(false)}
            className="fixed inset-0 z-40 cursor-default"
          />

          <div className="absolute right-0 top-full z-50 mt-2 w-[min(92vw,22rem)] rounded-2xl border border-white/10 arbor-popover p-4 shadow-2xl shadow-black/50 backdrop-blur-xl">
            <WalletConnect />
          </div>
        </>
      )}
    </div>
  );
}

function RoleBadge({ roles }: { roles: Roles }) {
  if (!roles.connected) return null;
  if (roles.isProtocolAuthority) {
    return (
      <span className="role-badge role-authority" title="Protocol authority (governance)">
        <span aria-hidden="true">◆</span>Authority
      </span>
    );
  }
  if (roles.oracleFor.length > 0) {
    return (
      <span className="role-badge role-oracle" title={"Authorized oracle submitter for " + roles.oracleFor.length + " market(s)"}>
        Oracle · {roles.oracleFor.length}
      </span>
    );
  }
  return (
    <span className="role-badge role-public" title="Public viewer">Public</span>
  );
}

export function Header() {
  const pathname = usePathname() ?? "";
  const [menuOpen, setMenuOpen] = useState(false);
  useEffect(() => { setMenuOpen(false); }, [pathname]);
  const roles = useRoles();
  const nav = NAV.filter((i) => i.href !== "/admin" || roles.isProtocolAuthority);

  return (
    <header className="sticky top-0 z-40 relative border-b border-white/5 arbor-surface">
      <div className="mx-auto max-w-6xl px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <img src="/logo-tree.svg" alt="ARBOR" className="arbor-mark h-9 w-auto shrink-0" />

            <div className="leading-tight">
              <p className="text-sm font-bold tracking-tight text-white">
                ARBOR
              </p>
              <p className="hidden text-[11px] text-zinc-500 sm:block">
                Canopy lending protocol
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2"><RoleBadge roles={roles} /><WalletChip /><button type="button" aria-label="Toggle menu" onClick={() => setMenuOpen((v) => !v)} className="grid h-9 w-9 place-items-center rounded-lg border border-white/10 text-zinc-200 transition hover:bg-white/5 md:hidden">{menuOpen ? "✕" : "☰"}</button></div>
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2">
          <StatusChip />

          <nav className="no-scrollbar -mx-1 hidden flex-1 items-center gap-1 overflow-x-auto px-1 md:flex">
            {nav.map((item) => {
              const active =
                item.href === "/"
                  ? pathname === "/"
                  : pathname.startsWith(item.href);

              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`whitespace-nowrap rounded-lg border px-3 py-1.5 text-xs transition ${
                    active
                      ? "border-white/10 bg-white/10 text-white"
                      : "border-transparent text-zinc-400 hover:bg-white/5 hover:text-zinc-200"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>
      </div>
    {menuOpen && (
        <>
          <button type="button" aria-label="Close menu" onClick={() => setMenuOpen(false)} className="arbor-scrim fixed inset-0 z-30 cursor-default md:hidden" />
          <div className="absolute left-0 right-0 top-full z-40 border-b border-white/10 arbor-surface-solid px-4 py-3 backdrop-blur-xl md:hidden">
            <nav className="grid gap-1">
              {nav.map((item) => {
                const active = item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
                return (
                  <Link key={item.href} href={item.href} onClick={() => setMenuOpen(false)} className={`rounded-lg border px-3 py-2 text-sm transition ${active ? "border-white/10 bg-white/10 text-white" : "border-transparent text-zinc-300 hover:bg-white/5"}`}>{item.label}</Link>
                );
              })}
            </nav>
          </div>
        </>
      )}
    </header>
  );
}
