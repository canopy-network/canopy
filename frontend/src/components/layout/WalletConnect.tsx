"use client";

import { useState } from "react";
import { adminGetKey } from "@/lib/canopy/rpc";
import { useWalletStore } from "@/lib/stores/walletStore";
import { publicKeyFromPrivateHex } from "@/lib/canopy/signing";
import { formatAddress } from "@/lib/arbor/format";

export function WalletConnect() {
  const { isConnected, address, connect, disconnect } = useWalletStore();

  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConnect(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const trimmed = identifier.trim();

    if (!trimmed) {
      setError("Enter wallet address or nickname.");
      return;
    }

    setBusy(true);

    try {
      const key = await adminGetKey(trimmed, password);

      if (!key) {
        throw new Error(
          "Admin RPC did not return a key. Check Canopy admin RPC and password."
        );
      }

      let publicKeyHex = key.publicKey;

      if (!publicKeyHex && key.privateKey) {
        publicKeyHex = publicKeyFromPrivateHex(key.privateKey);
      }

      if (!publicKeyHex) {
        throw new Error("No public key returned from admin RPC.");
      }

      if (!key.privateKey) {
        throw new Error("No private key returned from admin RPC.");
      }

      connect(trimmed, publicKeyHex, key.privateKey);

      setIdentifier("");
      setPassword("");
    } catch (err: any) {
      setError(err?.message || String(err));
    } finally {
      setBusy(false);
    }
  }

  if (isConnected) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur px-4 py-2">
        <span className="text-xs text-zinc-400">Wallet</span>
        <span className="font-mono text-xs text-zinc-200">
          {formatAddress(address || "")}
        </span>
        <button
          type="button"
          onClick={disconnect}
          className="rounded-lg border border-white/10 px-2 py-1 text-xs text-zinc-400 hover:text-zinc-200"
        >
          Disconnect
        </button>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleConnect}
      className="space-y-3 rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur p-4"
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <input
          value={identifier}
          onChange={(e) => setIdentifier(e.target.value)}
          placeholder="Wallet address or nickname"
          className="w-full rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
        />

        <input
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          type="password"
          placeholder="Keystore password (optional)"
          className="w-full rounded-xl border border-white/10 bg-white/[0.03] backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
        />
      </div>

      {error && <p className="text-xs text-rose-400">{error}</p>}

      <button
        type="submit"
        disabled={busy}
        className="inline-flex w-full items-center justify-center rounded-xl bg-gradient-to-r from-indigo-500 to-violet-500 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
      >
        {busy ? "Connecting..." : "Connect via Admin RPC"}
      </button>
    </form>
  );
}
