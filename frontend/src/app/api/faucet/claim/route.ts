import { NextRequest } from "next/server";
import { hexToBytes } from "@/lib/canopy/decode";
import { signAndSubmitArborTx } from "@/lib/canopy/submitArborTx";

/*
This route exists because the grad node's own plugin filesystem is
read-only (confirmed directly against a live error: opening
~/.canopy/keystore.json there returns errno 30 / EROFS). That means the
in-plugin pool faucet built in plugin/go/contract/faucet_pool.go — which
needs to decrypt a local keystore file to sign — cannot run there, even
though it works correctly on local devnet where the filesystem is
writable. faucetPoolHandleClaim in that file is UNCHANGED and still the
right approach for any deployment where the plugin's filesystem is
writable; this route is the workaround for the one where it isn't.

So signing happens HERE instead — a Vercel serverless function, holding
the pool's raw private key as an env var (same trust model as this
project's existing Discord/Twitter OAuth secrets), submitting directly to
the grad node's public core RPC using the SAME signing pipeline
(signAndSubmitArborTx / @noble/curves bls12_381) already used elsewhere in
this app for real deposit/borrow/repay transactions — not new, unproven
signing code.

Cooldown tracking still lives on the grad node's plugin
(faucetPoolHandleCheckAndReserve / faucetPoolHandleRollback in
faucet_pool.go) because that process CAN still read/write its own state
file — the read-only restriction only blocks the keystore path, not
faucet_pool_state.json (proven working end-to-end on local devnet before
the grad-node keystore issue was discovered). Those two endpoints do no
signing at all and never see the private key; they are a pure, shared,
persistent "did this address claim recently" ledger this route consults.

Flow:
  1. POST check-and-reserve to the plugin. If refused (cooldown), stop —
     nothing has been reserved, nothing to roll back.
  2. If reserved, sign and submit the actual send using the pool's key.
  3. If the send fails for any reason, POST rollback with the SAME height
     the reservation returned, so the address isn't left falsely locked
     out for a send that never happened.
  4. If the send succeeds, do nothing further — the reservation itself IS
     the record of a successful claim; there's no separate "confirm"
     endpoint by design (see the comment above faucetPoolHandleCheckAndReserve
     in faucet_pool.go for why).
*/

const PLUGIN_BASE =
  process.env.QUEST_PLUGIN_URL ||
  "https://arbor.val-a.grad.dev.app.canopynetwork.org/plugin";

const FAUCET_POOL_ADDRESS_HEX = process.env.FAUCET_POOL_ADDRESS_HEX || "";
const FAUCET_POOL_PUBLIC_KEY_HEX = process.env.FAUCET_POOL_PUBLIC_KEY_HEX || "";
const FAUCET_POOL_PRIVATE_KEY_HEX = process.env.FAUCET_POOL_PRIVATE_KEY_HEX || "";
const FAUCET_CLAIM_AMOUNT_UARB = BigInt(
  process.env.FAUCET_CLAIM_AMOUNT_UARB || "2000000" // 2 ARB default, matches faucet_pool.go's own default
);
const FAUCET_FEE_UARB = BigInt(process.env.FAUCET_FEE_UARB || "10000");
const ARBOR_CHAIN_ID = BigInt(process.env.ARBOR_CHAIN_ID || "1");
const ARBOR_NETWORK_ID = BigInt(process.env.ARBOR_NETWORK_ID || "1");

function isValidAddress(addr: string): boolean {
  return /^[0-9a-fA-F]{40}$/.test(addr);
}

async function pluginPost(path: string, body: unknown) {
  const res = await fetch(`${PLUGIN_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  const json = await res.json().catch(() => ({}));
  return { status: res.status, json };
}

export async function POST(req: NextRequest) {
  if (!FAUCET_POOL_ADDRESS_HEX || !FAUCET_POOL_PUBLIC_KEY_HEX || !FAUCET_POOL_PRIVATE_KEY_HEX) {
    return Response.json(
      { ok: false, error: "faucet not configured (FAUCET_POOL_*_HEX env vars unset)" },
      { status: 503 }
    );
  }

  let claimAddress: string;
  try {
    const body = await req.json();
    claimAddress = String(body?.address ?? "").toLowerCase().replace(/^0x/, "");
  } catch {
    return Response.json({ ok: false, error: "invalid JSON body" }, { status: 400 });
  }

  if (!isValidAddress(claimAddress)) {
    return Response.json(
      { ok: false, error: "address is not a well-formed 20-byte hex address" },
      { status: 400 }
    );
  }

  // Step 1: reserve the cooldown slot on the plugin's ledger BEFORE signing
  // or submitting anything. If this refuses, nothing has happened yet —
  // return its response as-is.
  const reserve = await pluginPost("/v1/faucetpool/check-and-reserve", {
    address: claimAddress,
  });
  if (reserve.status !== 200 || !reserve.json?.ok) {
    return Response.json(reserve.json, { status: reserve.status });
  }
  const reservedHeight = reserve.json.height;

  // Step 2: sign and submit the actual send, using the SAME pipeline this
  // app already uses for real deposit/borrow/repay transactions.
  try {
    const result = await signAndSubmitArborTx({
      txType: "send",
      msg: {
        fromAddress: hexToBytes(FAUCET_POOL_ADDRESS_HEX),
        toAddress: hexToBytes(claimAddress),
        amount: FAUCET_CLAIM_AMOUNT_UARB,
      },
      privateKeyHex: FAUCET_POOL_PRIVATE_KEY_HEX,
      publicKeyHex: FAUCET_POOL_PUBLIC_KEY_HEX,
      fee: FAUCET_FEE_UARB,
      memo: "arbor faucet",
      networkId: ARBOR_NETWORK_ID,
      chainId: ARBOR_CHAIN_ID,
    });

    return Response.json({
      ok: true,
      txHash: result.txHash,
      amountUarb: FAUCET_CLAIM_AMOUNT_UARB.toString(),
    });
  } catch (err) {
    // Step 3: the send failed after we'd already reserved — roll back so
    // this address isn't falsely locked out for a claim that never
    // actually happened.
    await pluginPost("/v1/faucetpool/rollback", {
      address: claimAddress,
      height: reservedHeight,
    }).catch(() => {
      // Best-effort: if even the rollback call fails (plugin unreachable,
      // etc.), the address stays reserved until the cooldown naturally
      // expires. Logged, not retried further -- see the tradeoff noted in
      // faucet_pool.go's comment above faucetPoolHandleCheckAndReserve.
      console.error("faucet rollback also failed after a send failure");
    });

    const message = err instanceof Error ? err.message : "send failed";
    return Response.json({ ok: false, error: message }, { status: 502 });
  }
}
