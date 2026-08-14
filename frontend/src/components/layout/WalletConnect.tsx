"use client";

import { useState, useEffect, useRef } from "react";
import { adminGetKey } from "@/lib/canopy/rpc";
import { useWalletStore } from "@/lib/wallet";
import {
  requestEthAccount,
  signDeriveMessage,
  deriveBlsFromEthSignature,
  cacheDerivedKey,
  getAlreadyConnectedEthAccount,
  loadCachedKey,
  hasCachedKey,
} from "@/lib/wallet/metamask";
import {
  cachePrivateKey,
  loadCachedPrivateKey,
  hasCachedPrivateKey,
  lastConnectedAddress,
} from "@/lib/wallet/deviceCache";
import { publicKeyFromPrivateHex } from "@/lib/wallet";
import { formatAddress } from "@/lib/arbor/format";


function errMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}

type ConnectTab = "generate" | "paste" | "import" | "admin" | "metamask";

export function WalletConnect() {
  const {
    isConnected,
    address,
    disconnect,
    hasStoredKeystore,
    saveKeystore,
    loadFromKeystore,
    clearStoredKeystore,
    connect,
    generateAndDownloadKeystore,
    connectFromRawKey,
    importFromKeystoreFile,
    privateKeyHex,
  } = useWalletStore();

  const [tab, setTab] = useState<ConnectTab>("generate");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dismissedUnlock, setDismissedUnlock] = useState(false);
  const showUnlock = !isConnected && hasStoredKeystore && !dismissedUnlock;

  // Silent reconnect on load: device-cache first (any connect method),
  // then MetaMask silent (already connected + cached derived key).
  useEffect(() => {
    if (isConnected) return;
    let alive = true;
    (async () => {
      // 1) Device-cache (AES-GCM encrypted, works for all connect methods)
      const lastAddr = lastConnectedAddress();
      if (lastAddr && hasCachedPrivateKey(lastAddr)) {
        const key = await loadCachedPrivateKey(lastAddr);
        if (key && alive) {
          await connectFromRawKey(key);
          return;
        }
      }
      // 2) MetaMask silent (already connected + cached derived key)
      const eth = await getAlreadyConnectedEthAccount();
      if (!eth || !hasCachedKey(eth)) return;
      const mmKey = await loadCachedKey(eth);
      if (!mmKey || !alive) return;
      await connectFromRawKey(mmKey);
    })().catch(() => {});
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isConnected]);

  // Generate tab
  const [genPassword, setGenPassword] = useState("");
  const [genPassword2, setGenPassword2] = useState("");
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [downloadedFilename, setDownloadedFilename] = useState<string | null>(null);
  const [ackBackup, setAckBackup] = useState(false);

  // Paste tab
  const [pastedKey, setPastedKey] = useState("");

  // Import tab
  const [importPassword, setImportPassword] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Admin RPC tab (legacy/dev path)
  const [identifier, setIdentifier] = useState("");
  const [adminPassword, setAdminPassword] = useState("");

  // Shared "remember this wallet" keystore controls
  const [rememberWallet, setRememberWallet] = useState(false);
  const [keystorePassword, setKeystorePassword] = useState("");

  async function maybeSaveKeystore() {
    if (rememberWallet && address && privateKeyHex) {
      // Device-cache (AES-GCM encrypted, auto-reconnect without prompt)
      await cachePrivateKey(address, privateKeyHex);
      // Password keystore (requires unlock prompt on revisit)
      if (keystorePassword) {
        if (keystorePassword.length < 8) {
          throw new Error("Keystore password must be at least 8 characters");
        }
        await saveKeystore(keystorePassword);
      }
    }
    setKeystorePassword("");
    setRememberWallet(false);
  }

  async function handleGenerate(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (genPassword !== genPassword2) {
      setError("Passwords do not match.");
      return;
    }
    if (genPassword.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    setBusy(true);
    try {
      const { privateKeyHex: pk, keystoreJson, filename } = await generateAndDownloadKeystore(genPassword);

      const blob = new Blob([keystoreJson], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);

      setGeneratedKey(pk);
      setDownloadedFilename(filename);
      setGenPassword("");
      setGenPassword2("");
    } catch (err: unknown) {
      setError(errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  function handleConfirmGenerated() {
    setError(null);
    if (!ackBackup) {
      setError("Confirm you've backed up the private key before continuing.");
      return;
    }
    setGeneratedKey(null);
    setDownloadedFilename(null);
    setAckBackup(false);
  }

  async function handlePaste(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await connectFromRawKey(pastedKey);
      setPastedKey("");
      await maybeSaveKeystore();
    } catch (err: unknown) {
      setError(errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleImport(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      setError("Select a keystore file.");
      return;
    }
    if (!importPassword) {
      setError("Enter the keystore file's password.");
      return;
    }
    setBusy(true);
    try {
      const text = await file.text();
      const raw = JSON.parse(text);
      await importFromKeystoreFile(raw, importPassword);
      setImportPassword("");
      if (fileInputRef.current) fileInputRef.current.value = "";
    } catch (err: unknown) {
      setError(errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleMetaMaskConnect() {
    setError(null);
    setBusy(true);
    try {
      const ethAddress = await requestEthAccount();
      setError(null);
      const sig = await signDeriveMessage(ethAddress);
      const derived = await deriveBlsFromEthSignature(sig);
      await connectFromRawKey(derived.privateKeyHex);
      await cacheDerivedKey(ethAddress, derived.privateKeyHex);
      await maybeSaveKeystore();
    } catch (err) {
      setError(errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleAdminConnect(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const trimmed = identifier.trim();
    if (!trimmed) {
      setError("Enter wallet address or nickname.");
      return;
    }

    setBusy(true);

    try {
      const key = await adminGetKey(trimmed, adminPassword);
      if (!key) {
        throw new Error(
          "Admin RPC did not return a key. Check Canopy admin RPC and password."
        );
      }

      let publicKeyHex = key.publicKey;
      if (!publicKeyHex && key.privateKey) {
        publicKeyHex = publicKeyFromPrivateHex(key.privateKey);
      }
      if (!publicKeyHex) throw new Error("No public key returned from admin RPC.");
      if (!key.privateKey) throw new Error("No private key returned from admin RPC.");

      connect(trimmed, publicKeyHex, key.privateKey);
      await maybeSaveKeystore();

      setIdentifier("");
      setAdminPassword("");
    } catch (err: unknown) {
      setError(errMessage(err));
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
      setDismissedUnlock(false);
    } catch (err: unknown) {
      setError(errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  function handleClearKeystore() {
    clearStoredKeystore();
    setDismissedUnlock(false);
    setError(null);
  }

  const inputClass =
    "w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-500/20";
  const primaryBtnClass =
    "inline-flex items-center justify-center rounded-xl btn-brand px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none";
  const secondaryBtnClass =
    "rounded-xl border border-white/10 px-4 py-2.5 text-sm text-zinc-400 hover:text-zinc-200";

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
        <button type="button" onClick={() => { setDismissedUnlock(false); disconnect(); }} className="rounded-lg border border-white/10 px-2 py-1 text-xs text-zinc-400 hover:text-zinc-200">
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
            className={inputClass}
            autoFocus
          />
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <div className="flex gap-2">
            <button type="submit" disabled={busy} className={`flex-1 ${primaryBtnClass}`}>
              {busy ? "Unlocking..." : "Unlock"}
            </button>
            <button type="button" onClick={handleClearKeystore} className={secondaryBtnClass}>
              Clear
            </button>
            <button type="button" onClick={() => setDismissedUnlock(true)} className={secondaryBtnClass}>
              New
            </button>
          </div>
        </form>
      </div>
    );
  }

  if (generatedKey) {
    return (
      <div className="space-y-3 rounded-xl glass backdrop-blur p-4">
        <div className="text-sm text-emerald-400">
          ✓ Keystore file downloaded{downloadedFilename ? `: ${downloadedFilename}` : ""}
        </div>
        <div className="text-sm text-zinc-300">
          Also back up the raw private key below somewhere safe — it is never
          sent anywhere and cannot be recovered if lost.
        </div>
        <div className="space-y-1">
          <div className="text-xs text-zinc-400">Address</div>
          <div className="font-mono text-xs text-zinc-200 break-all">{formatAddress(address || "")}</div>
        </div>
        <div className="space-y-1">
          <div className="text-xs text-zinc-400">Private key (keep secret)</div>
          <div className="font-mono text-xs text-amber-300 break-all rounded-lg border border-white/5 bg-white/[0.02] p-2">
            {generatedKey}
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm text-zinc-300">
          <input type="checkbox" checked={ackBackup} onChange={(e) => setAckBackup(e.target.checked)} className="rounded border-white/20" />
          I&apos;ve backed up the keystore file and/or private key
        </label>
        {error && <p className="text-xs text-rose-400">{error}</p>}
        <button type="button" disabled={!ackBackup} onClick={handleConfirmGenerated} className={`w-full ${primaryBtnClass}`}>
          Continue
        </button>
      </div>
    );
  }

  const tabs: { id: ConnectTab; label: string }[] = [
    { id: "metamask", label: "MetaMask" },
    { id: "generate", label: "New wallet" },
    { id: "paste", label: "Paste key" },
    { id: "import", label: "Import file" },
    { id: "admin", label: "Admin RPC" },
  ];

  return (
    <div className="space-y-3 rounded-xl glass backdrop-blur p-4">
      <div className="flex gap-1 rounded-lg border border-white/5 bg-white/[0.02] p-1">
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => {
              setTab(t.id);
              setError(null);
            }}
            className={`flex-1 rounded-md px-2 py-1.5 text-xs font-medium transition ${
              tab === t.id ? "bg-indigo-500/20 text-indigo-200" : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "generate" && (
        <form onSubmit={handleGenerate} className="space-y-3">
          <p className="text-xs text-zinc-400">
            Generates a new BLS12-381 keypair entirely in your browser and
            downloads a password-encrypted keystore file. The private key
            never leaves this device.
          </p>
          <input
            value={genPassword}
            onChange={(e) => setGenPassword(e.target.value)}
            type="password"
            placeholder="Keystore password (min 8 chars)"
            className={inputClass}
          />
          <input
            value={genPassword2}
            onChange={(e) => setGenPassword2(e.target.value)}
            type="password"
            placeholder="Confirm password"
            className={inputClass}
          />
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <button type="submit" disabled={busy} className={`w-full ${primaryBtnClass}`}>
            {busy ? "Generating..." : "Generate & download keystore"}
          </button>
        </form>
      )}

      {tab === "paste" && (
        <form onSubmit={handlePaste} className="space-y-3">
          <input
            value={pastedKey}
            onChange={(e) => setPastedKey(e.target.value)}
            placeholder="Private key (64 hex chars)"
            className={`${inputClass} font-mono`}
          />
          <div className="space-y-2 rounded-lg border border-white/5 bg-white/[0.02] p-3">
            <label className="flex items-center gap-2 text-sm text-zinc-300">
              <input type="checkbox" checked={rememberWallet} onChange={(e) => setRememberWallet(e.target.checked)} className="rounded border-white/20" />
              Remember this wallet (encrypted local storage)
            </label>
            {rememberWallet && (
              <input
                value={keystorePassword}
                onChange={(e) => setKeystorePassword(e.target.value)}
                type="password"
                placeholder="Keystore password (min 8 chars)"
                className={inputClass}
              />
            )}
          </div>
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <button type="submit" disabled={busy} className={`w-full ${primaryBtnClass}`}>
            {busy ? "Connecting..." : "Connect"}
          </button>
        </form>
      )}

      {tab === "import" && (
        <form onSubmit={handleImport} className="space-y-3">
          <p className="text-xs text-zinc-400">
            Accepts a keystore file from this app, Praxis, or the Canopy CLI.
          </p>
          <input ref={fileInputRef} type="file" accept="application/json,.json" className={inputClass} />
          <input
            value={importPassword}
            onChange={(e) => setImportPassword(e.target.value)}
            type="password"
            placeholder="Keystore file password"
            className={inputClass}
          />
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <button type="submit" disabled={busy} className={`w-full ${primaryBtnClass}`}>
            {busy ? "Importing..." : "Import"}
          </button>
        </form>
      )}

      {tab === "metamask" && (
        <div className="space-y-3">
          <p className="text-xs text-zinc-400">
            Connect using MetaMask. Signs a derivation message to generate a
            deterministic BLS key from your Ethereum account.
          </p>
          <div className="space-y-2 rounded-lg border border-white/5 bg-white/[0.02] p-3">
            <label className="flex items-center gap-2 text-sm text-zinc-300">
              <input type="checkbox" checked={rememberWallet} onChange={(e) => setRememberWallet(e.target.checked)} className="rounded border-white/20" />
              Remember this wallet (encrypted local storage)
            </label>
            {rememberWallet && (
              <input
                value={keystorePassword}
                onChange={(e) => setKeystorePassword(e.target.value)}
                type="password"
                placeholder="Keystore password (min 8 chars)"
                className={inputClass}
              />
            )}
          </div>
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <button type="button" onClick={handleMetaMaskConnect} disabled={busy} className={`w-full ${primaryBtnClass}`}>
            {busy ? "Connecting..." : "Connect with MetaMask"}
          </button>
        </div>
      )}

      {tab === "admin" && (
        <form onSubmit={handleAdminConnect} className="space-y-3">
          <p className="text-xs text-zinc-400">
            Fetches your private key from the Canopy node&apos;s admin RPC. Only
            use this against a node you trust — it sees your key in plaintext.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            <input
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              placeholder="Wallet address or nickname"
              className={inputClass}
            />
            <input
              value={adminPassword}
              onChange={(e) => setAdminPassword(e.target.value)}
              type="password"
              placeholder="Canopy admin password"
              className={inputClass}
            />
          </div>
          <div className="space-y-2 rounded-lg border border-white/5 bg-white/[0.02] p-3">
            <label className="flex items-center gap-2 text-sm text-zinc-300">
              <input type="checkbox" checked={rememberWallet} onChange={(e) => setRememberWallet(e.target.checked)} className="rounded border-white/20" />
              Remember this wallet (encrypted local storage)
            </label>
            {rememberWallet && (
              <input
                value={keystorePassword}
                onChange={(e) => setKeystorePassword(e.target.value)}
                type="password"
                placeholder="Keystore password (min 8 chars)"
                className={inputClass}
              />
            )}
          </div>
          {error && <p className="text-xs text-rose-400">{error}</p>}
          <button type="submit" disabled={busy} className={`w-full ${primaryBtnClass}`}>
            {busy ? "Connecting..." : "Connect via Admin RPC"}
          </button>
        </form>
      )}
    </div>
  );
}
