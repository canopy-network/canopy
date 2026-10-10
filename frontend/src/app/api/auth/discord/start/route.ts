import { NextRequest, NextResponse } from "next/server";
import { randomBytes } from "node:crypto";

/**
 * Kicks off Discord OAuth. Requires DISCORD_CLIENT_ID and APP_URL env vars
 * (see .env.example). A random `state` value is stored in a short-lived
 * cookie and echoed back by Discord on callback — standard CSRF protection
 * for OAuth redirects, so a third party can't trick a user's browser into
 * completing a login flow the user didn't start.
 */
export async function GET() {
  const state = randomBytes(16).toString("hex");
  const appUrl = process.env.APP_URL ?? "http://localhost:3000";

  const params = new URLSearchParams({
    client_id: process.env.DISCORD_CLIENT_ID ?? "",
    redirect_uri: `${appUrl}/api/auth/discord/callback`,
    response_type: "code",
    scope: "identify",
    state,
  });

  const res = NextResponse.redirect(`https://discord.com/api/oauth2/authorize?${params}`);
  res.cookies.set("discord_oauth_state", state, { httpOnly: true, maxAge: 600, path: "/" });
  return res;
}
