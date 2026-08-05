"use client";

import { useState, useEffect } from "react";
import { adminGetKey } from "@/lib/canopy/rpc";
import { useWalletStore } from "@/lib/wallet";
import { publicKeyFromPrivateHex } from "@/lib/wallet";
import { formatAddress } from "@/lib/arbor/format";

export function WalletConnect() {
  const {
    isConnected,
    address,
    connect,
    disconnect,
    hasStoredKeystore,
    saveKeystore,
    loadFromKeystore,
    clearStoredKeystore,
  } = useWalletStore();

  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [keystorePassword, setKeystorePassword] = useState("");
  const [rememberWallet, setRememberWallet] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showUnlock, setShowUnlock] = useState(false);

  // Show unlock option if keystore exists and wallet is not connected
  useEffect(() => {
    if (!isConnected && hasStoredKeystore) {
      setShowUnlock(true);
    }
  }, [isConnected, hasStoredKeystore]);

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

      // Save to keystore if requested
      if (rememberWallet && keystorePassword) {
        if (keystorePassword.length < 8) {
          throw new Error("Keystore password must be at least 8 characters");
        }
        await saveKeystore(keystorePassword);
      }

      setIdentifier("");
      setPassword("");
      setKeystorePassword("");
      setRememberWallet(false);
    } catch (err: any) {
      setError(err?.message || String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleUnlock(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!keystorePassword) {
      setError("Enter keystore password");
      return;
    }

    setBusy(true);

    try {
      await loadFromKeystore(keystorePassword);
      setKeystorePassword("");
      setShowUnlock(false);
    } catch (err: any) {
      setError(err?.message || String(err));
    } finally {
      setBusy(false);
    }
  }

  function handleClearKeystore() {
    clearStoredKeystore();
    setShowUnlock(false);
    setError(null);
  }

  if (isConnected) {
    return (
      <div className="flex items-center gap-3 rounded-xl glass backdrop-blur px-4 py-2">
        <span className="text-xs text-zinc-400">Wallet</span>
        <span className="font-mono text-xs text-zinc-200">
          {formatAddress(address || "")}
        </span>
        {hasStoredKeystore && (
          <span className="text-xs text-emerald-400" title="Saved to encrypted keystore">
            🔒
          </span>
        )}
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

  if (showUnlock && hasStoredKeystore) {
    return (
      <div className="space-y-3 rounded-xl glass backdrop-blur p-4">
        <div className="text-sm text-zinc-300">
          Saved wallet detected. Enter keystore password to unlock:
        </div>
        <form onSubmit={handleUnlock} className="space-y-3">
          <input
            value={keystorePassword}
            onChange={(e) => setKeystorePassword(e.target.value)}
            type="password"
            placeholder="Keystore password"
            className="w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
            autoFocus
          />
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={busy}
              className="flex-1 inline-flex items-center justify-center rounded-xl btn-brand px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
            >
              {busy ? "Unlocking..." : "Unlock"}
            </button>
            <button
              type="button"
              onClick={handleClearKeystore}
              className="rounded-xl border border-white/10 px-4 py-2.5 text-sm text-zinc-400 hover:text-zinc-200"
            >
              Clear
            </button>
            <button
              type="button"
              onClick={() => setShowUnlock(false)}
              className="rounded-xl border border-white/10 px-4 py-2.5 text-sm text-zinc-400 hover:text-zinc-200"
            >
              New
            </button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleConnect}
      className="space-y-3 rounded-xl glass backdrop-blur p-4"
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <input
          value={identifier}
          onChange={(e) => setIdentifier(e.target.value)}
          placeholder="Wallet address or nickname"
          className="w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
        />
        <input
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          type="password"
          placeholder="Canopy admin password"
          className="w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
        />
      </div>

      <div className="space-y-2 rounded-lg border border-white/5 bg-white/[0.02] p-3">
        <label className="flex items-center gap-2 text-sm text-zinc-300">
          <input
            type="checkbox"
            checked={rememberWallet}
            onChange={(e) => setRememberWallet(e.target.checked)}
            className="rounded border-white/20"
          />
          Remember this wallet (encrypted local storage)
        </label>
        {rememberWallet && (
          <input
            value={keystorePassword}
            onChange={(e) => setKeystorePassword(e.target.value)}
            type="password"
            placeholder="Keystore password (min 8 chars)"
            className="w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20"
          />
        )}
      </div>

      {error && <p className="text-xs text-rose-400">{error}</p>}

      <button
        type="submit"
        disabled={busy}
        className="inline-flex w-full items-center justify-center rounded-xl btn-brand px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none"
      >
        {busy ? "Connecting..." : "Connect via Admin RPC"}
      </button>
    </form>
  );
}
