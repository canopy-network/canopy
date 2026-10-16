import { NextRequest, NextResponse } from "next/server";

/**
 * Discord's redirect target. Exchanges the auth code for an access token,
 * fetches the user's Discord ID, and stores it in a short-lived cookie for
 * the /link page to read. Nothing is written to the indexer here — this
 * route only proves "this browser owns this Discord account"; the actual
 * wallet-signature binding happens later, on /link.
 */
export async function GET(req: NextRequest) {
  const appUrl = process.env.APP_URL ?? "http://localhost:3000";
  const url = new URL(req.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const expectedState = req.cookies.get("discord_oauth_state")?.value;

  if (!code || !state || !expectedState || state !== expectedState) {
    return NextResponse.redirect(`${appUrl}/quests?error=discord_state_mismatch`);
  }

  const tokenRes = await fetch("https://discord.com/api/oauth2/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      client_id: process.env.DISCORD_CLIENT_ID ?? "",
      client_secret: process.env.DISCORD_CLIENT_SECRET ?? "",
      grant_type: "authorization_code",
      code,
      redirect_uri: `${appUrl}/api/auth/discord/callback`,
    }),
  });

  if (!tokenRes.ok) {
    return NextResponse.redirect(`${appUrl}/quests?error=discord_token_exchange_failed`);
  }

  const { access_token } = await tokenRes.json();

  const userRes = await fetch("https://discord.com/api/users/@me", {
    headers: { Authorization: `Bearer ${access_token}` },
  });

  if (!userRes.ok) {
    return NextResponse.redirect(`${appUrl}/quests?error=discord_user_fetch_failed`);
  }

  const discordUser = await userRes.json();

  const res = NextResponse.redirect(`${appUrl}/quests?linked=discord`);
  res.cookies.set("discord_id", discordUser.id, { httpOnly: true, maxAge: 3600, path: "/" });
  res.cookies.set("discord_username", discordUser.username ?? "", { httpOnly: true, maxAge: 3600, path: "/" });
  res.cookies.delete("discord_oauth_state");
  return res;
}
