import { NextResponse } from "next/server";
import { randomBytes, createHash } from "node:crypto";

/**
 * X's OAuth 2.0 requires PKCE (a code_verifier/code_challenge pair) even
 * for confidential clients. The verifier is stored in a cookie and sent
 * again on callback to prove this is the same browser session that
 * started the flow — a second CSRF-style protection alongside `state`.
 * Requires TWITTER_CLIENT_ID and APP_URL env vars.
 */
function base64url(input: Buffer): string {
  return input.toString("base64").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export async function GET() {
  const appUrl = process.env.APP_URL ?? "http://localhost:3000";
  const state = randomBytes(16).toString("hex");
  const codeVerifier = base64url(randomBytes(32));
  const codeChallenge = base64url(createHash("sha256").update(codeVerifier).digest());

  const params = new URLSearchParams({
    response_type: "code",
    client_id: process.env.TWITTER_CLIENT_ID ?? "",
    redirect_uri: `${appUrl}/api/auth/twitter/callback`,
    scope: "users.read tweet.read",
    state,
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
  });

  const res = NextResponse.redirect(`https://twitter.com/i/oauth2/authorize?${params}`);
  res.cookies.set("twitter_oauth_state", state, { httpOnly: true, maxAge: 600, path: "/" });
  res.cookies.set("twitter_code_verifier", codeVerifier, { httpOnly: true, maxAge: 600, path: "/" });
  return res;
}
