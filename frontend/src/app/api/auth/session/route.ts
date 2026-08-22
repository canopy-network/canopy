import { NextRequest, NextResponse } from "next/server";

/**
 * Cookies set by the OAuth callbacks are httpOnly (can't be read from
 * client-side JS directly, which is deliberate — keeps the raw Discord ID
 * out of reach of any injected script). This route is the one sanctioned
 * way for the /link page to check "what's already connected in this
 * browser session" without exposing the cookies themselves.
 */
export async function GET(req: NextRequest) {
  const discordId = req.cookies.get("discord_id")?.value ?? null;
  const discordUsername = req.cookies.get("discord_username")?.value ?? null;
  const twitterHandle = req.cookies.get("twitter_handle")?.value ?? null;
  return NextResponse.json({ discordId, discordUsername, twitterHandle });
}
