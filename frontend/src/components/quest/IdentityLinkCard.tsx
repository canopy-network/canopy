"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useWalletStore } from "@/lib/wallet";
import { signBls12381, bytesToHex } from "@/lib/wallet/signer";
import { queryHeight } from "@/lib/canopy/rpc";
import { linkIdentity, fetchIdentity, type IdentityResponse } from "@/lib/quest/api";
import {
  requestEthAccount,
  getAlreadyConnectedEthAccount,
  lastConnectedEthAddress,
} from "@/lib/wallet/metamask";

const OAUTH_ERROR_MESSAGES: Record<string, string> = {
  discord_state_mismatch: "Discord connection expired or was tampered with — please try again.",
  discord_token_exchange_failed: "Discord rejected the connection. This usually means DISCORD_CLIENT_ID/DISCORD_CLIENT_SECRET are missing or wrong in the deployment config.",
  discord_user_fetch_failed: "Connected to Discord but couldn't fetch your profile — try again.",
  twitter_state_mismatch: "X connection expired or was tampered with — please try again.",
  twitter_not_configured: "X login isn't configured on this deployment yet (TWITTER_CLIENT_ID is missing).",
  twitter_token_exchange_failed: "X rejected the connection. This usually means TWITTER_CLIENT_ID/TWITTER_CLIENT_SECRET are missing or wrong in the deployment config, or the redirect URI isn't registered in the X Developer Portal.",
  twitter_user_fetch_failed: "Connected to X but couldn't fetch your profile — try again.",
};

function shortAddr(addr: string): string {
  return addr.length > 10 ? `${addr.slice(0, 6)}...${addr.slice(-4)}` : addr;
}

interface SessionStatus {
  discordId: string | null;
  discordUsername: string | null;
  twitterHandle: string | null;
}

interface IdentityLinkCardProps {
  /** Card gets a slightly different intro line depending on where it's mounted. */
  variant?: "quests" | "standalone";
  onLinked?: () => void;
}

export function IdentityLinkCard({ variant = "quests", onLinked }: IdentityLinkCardProps) {
  const { address, publicKeyHex, privateKeyHex, isConnected } = useWalletStore();
  const searchParams = useSearchParams();
  const [session, setSession] = useState<SessionStatus | null>(null);
  const [evmAddress, setEvmAddress] = useState<string | null>(null);
  const [evmConnecting, setEvmConnecting] = useState(false);
  const [linking, setLinking] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null);
  // "checking" until the lookup resolves, so we never flash the full OAuth flow for an
  // address that's actually already linked. "linked"/"unlinked" once we know.
  const [identityStatus, setIdentityStatus] = useState<"idle" | "checking" | "linked" | "unlinked">("idle");
  const [linkedIdentity, setLinkedIdentity] = useState<IdentityResponse | null>(null);

  // The actual fix for "I have to re-authorize every reload": identity linking is a
  // one-time binding of Discord/X/EVM to an Arbor address, permanently recorded server-side
  // by /v1/link. There was previously no way to ask the backend "is this address already
  // linked" — every reconnect re-ran the full OAuth+sign flow even for an address that had
  // already linked. This looks it up on wallet connect (or address change) and, if linked,
  // shows the bound identity read-only instead of the interactive flow.
  useEffect(() => {
    if (!address) {
      setIdentityStatus("idle");
      setLinkedIdentity(null);
      return;
    }
    let alive = true;
    setIdentityStatus("checking");
    fetchIdentity(address).then((res) => {
      if (!alive) return;
      if (res?.linked) {
        setLinkedIdentity(res);
        setIdentityStatus("linked");
      } else {
        setLinkedIdentity(null);
        setIdentityStatus("unlinked");
      }
    });
    return () => {
      alive = false;
    };
  }, [address]);

  // Surfaces the ?error=/?linked= params the Discord/Twitter OAuth callback routes set.
  // Nothing previously read these — an OAuth failure (bad client secret, unregistered
  // redirect URI, etc.) redirected back here with an error code in the URL that was
  // silently dropped, indistinguishable from the user never having clicked anything.
  useEffect(() => {
    const errorCode = searchParams.get("error");
    const linked = searchParams.get("linked");
    if (errorCode) {
      setResult({ ok: false, message: OAUTH_ERROR_MESSAGES[errorCode] ?? `Connection failed (${errorCode}).` });
    } else if (linked) {
      setResult({ ok: true, message: `${linked === "discord" ? "Discord" : "X"} connected.` });
    }
    // Intentionally runs once on mount only — searchParams is stable for a given page load,
    // and we don't want this re-firing on every unrelated re-render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    fetch("/api/auth/session")
      .then((r) => r.json())
      .then(setSession)
      .catch(() => setSession({ discordId: null, discordUsername: null, twitterHandle: null }));
  }, []);

  // Silent EVM address pickup: if the user already connected MetaMask this session
  // (for wallet derivation, or previously), reuse that address without prompting for
  // another signature. requestEthAccount() below is only for the explicit "Connect" click.
  useEffect(() => {
    let alive = true;
    (async () => {
      const cached = lastConnectedEthAddress();
      if (cached && alive) {
        setEvmAddress(cached);
        return;
      }
      const already = await getAlreadyConnectedEthAccount();
      if (already && alive) setEvmAddress(already);
    })();
    return () => {
      alive = false;
    };
  }, []);

  async function handleConnectEvm() {
    setEvmConnecting(true);
    setResult(null);
    try {
      const acct = await requestEthAccount();
      setEvmAddress(acct);
    } catch (err) {
      setResult({ ok: false, message: err instanceof Error ? err.message : "Could not connect MetaMask." });
    } finally {
      setEvmConnecting(false);
    }
  }

  const discordDone = !!session?.discordId;
  const twitterDone = !!session?.twitterHandle;
  // EVM address is optional (see quest_xp.go NOTE on the mandatory/optional decision) —
  // not included in readyToLink, so linking is never blocked on it.
  const readyToLink = isConnected && discordDone && twitterDone;

  async function handleLink() {
    if (!address || !publicKeyHex || !privateKeyHex || !session?.discordId || !session?.twitterHandle) return;
    setLinking(true);
    setResult(null);
    try {
      const height = await queryHeight();
      if (!height) throw new Error("Could not reach Arbor node to fetch current height");

      // Message format must exactly match quest_xp.go's questXPHandleLink — including
      // the "none" literal for an absent EVM address, so the signature stays valid
      // whether or not MetaMask is connected.
      // Mirror backend normalization (quest_xp.go questXPHandleLink):
      // - empty/whitespace → "none"
      // - non-empty → lowercase the whole 0x-prefixed string
      const evmForMessage = (!evmAddress || !evmAddress.trim()) 
        ? "none" 
        : evmAddress.toLowerCase();
      const message = `Link Arbor identity\ndiscord:${session.discordId}\ntwitter:${session.twitterHandle}\nevm:${evmForMessage}\nissuedAt:${height}`;
      const signature = signBls12381(new TextEncoder().encode(message), privateKeyHex);
      const signatureHex = bytesToHex(signature);

      const res = await linkIdentity({
        address,
        publicKeyHex,
        discordId: session.discordId,
        twitterHandle: session.twitterHandle,
        ...(evmForMessage !== "none" ? { evmAddress: evmForMessage } : {}),
        issuedAtHeight: height,
        signatureHex,
      });

      if (res.ok) {
        setResult({ ok: true, message: "Linked! Your quest completions will now be tracked." });
        const fresh = await fetchIdentity(address);
        if (fresh?.linked) {
          setLinkedIdentity(fresh);
          setIdentityStatus("linked");
        }
        onLinked?.();
      } else {
        setResult({ ok: false, message: res.error ?? "Linking failed." });
      }
    } catch (err) {
      setResult({ ok: false, message: err instanceof Error ? err.message : "Something went wrong." });
    } finally {
      setLinking(false);
    }
  }

  function formatLinkedAt(ms?: number): string {
    if (!ms) return "";
    return new Date(ms).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
  }

  return (
    <div
      className={
        variant === "quests"
          ? "mb-8 rounded-lg border border-neutral-800 bg-neutral-900/50 p-6"
          : ""
      }
    >
      {identityStatus === "linked" && linkedIdentity ? (
        <>
          <div className="mb-1 text-sm text-neutral-400">Identity linked</div>
          <p className="mb-4 text-xs text-neutral-600">
            This Arbor address is permanently bound to the credentials below
            {linkedIdentity.linkedAt ? ` as of ${formatLinkedAt(linkedIdentity.linkedAt)}` : ""}. No need to
            reconnect Discord or X again — this only needs doing once per address.
          </p>
          <div className="space-y-3">
            <div className="rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div className="text-sm font-medium">Wallet</div>
              <div className="text-xs text-neutral-500">{shortAddr(address ?? linkedIdentity.address ?? "")}</div>
            </div>
            <div className="rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div className="text-sm font-medium">Discord</div>
              <div className="text-xs text-emerald-400">Linked</div>
            </div>
            <div className="rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div className="text-sm font-medium">X (Twitter)</div>
              <div className="text-xs text-emerald-400">Linked</div>
            </div>
            <div className="rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div className="text-sm font-medium">
                EVM address <span className="font-normal text-neutral-600">(optional)</span>
              </div>
              <div className="text-xs text-neutral-500">
                {linkedIdentity.evmAddress ? shortAddr(linkedIdentity.evmAddress) : "Not linked"}
              </div>
            </div>
          </div>
        </>
      ) : identityStatus === "checking" ? (
        <>
          <div className="mb-1 text-sm text-neutral-400">Link your identity</div>
          <div className="animate-pulse text-xs text-neutral-600">Checking whether this address is already linked…</div>
        </>
      ) : (
        <>
          <div className="mb-1 text-sm text-neutral-400">Link your identity</div>
          <p className="mb-4 text-xs text-neutral-600">
            Link Discord and X to your wallet so quest XP is attributed to you. Connecting
            MetaMask is optional but recommended — it's how reward distribution will reach you.
            One-time signature — your key never leaves the browser, and this only needs doing
            once per address.
          </p>

          <div className="space-y-3">
            <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div>
                <div className="text-sm font-medium">Wallet</div>
                <div className="text-xs text-neutral-500">
                  {isConnected ? shortAddr(address ?? "") : "Not connected"}
                </div>
              </div>
              {!isConnected && <span className="text-xs text-amber-400">Connect wallet first</span>}
            </div>

            <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div>
                <div className="text-sm font-medium">Discord</div>
                <div className="text-xs text-neutral-500">
                  {discordDone ? `Connected as ${session?.discordUsername}` : "Not connected"}
                </div>
              </div>
              {!discordDone && (
                <a
                  href="/api/auth/discord/start"
                  className="rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-500"
                >
                  Connect
                </a>
              )}
            </div>

            <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div>
                <div className="text-sm font-medium">X (Twitter)</div>
                <div className="text-xs text-neutral-500">
                  {twitterDone ? `Connected as ${session?.twitterHandle}` : "Not connected"}
                </div>
              </div>
              {!twitterDone && (
                <a
                  href="/api/auth/twitter/start"
                  className="rounded-md bg-neutral-100 px-3 py-1.5 text-xs font-medium text-neutral-900 hover:bg-white"
                >
                  Connect
                </a>
              )}
            </div>

            <div className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950/50 p-4">
              <div>
                <div className="text-sm font-medium">
                  EVM address <span className="font-normal text-neutral-600">(optional)</span>
                </div>
                <div className="text-xs text-neutral-500">
                  {evmAddress ? shortAddr(evmAddress) : "Not connected"}
                </div>
              </div>
              {!evmAddress && (
                <button
                  type="button"
                  onClick={handleConnectEvm}
                  disabled={evmConnecting}
                  className="rounded-md bg-orange-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-400 disabled:opacity-40"
                >
                  {evmConnecting ? "Connecting…" : "Connect"}
                </button>
              )}
            </div>
          </div>

          <button
            onClick={handleLink}
            disabled={!readyToLink || linking}
            className="mt-4 w-full rounded-lg bg-gradient-to-r from-teal-500 to-indigo-600 py-2.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
          >
            {linking ? "Signing…" : "Sign & Link"}
          </button>
        </>
      )}

    </div>
  );
}
