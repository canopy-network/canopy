import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const appUrl = process.env.APP_URL ?? "http://localhost:3000";
  const url = new URL(req.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const expectedState = req.cookies.get("twitter_oauth_state")?.value;
  const codeVerifier = req.cookies.get("twitter_code_verifier")?.value;

  if (!code || !state || !expectedState || state !== expectedState || !codeVerifier) {
    return NextResponse.redirect(`${appUrl}/link?error=twitter_state_mismatch`);
  }

  const basicAuth = Buffer.from(
    `${process.env.TWITTER_CLIENT_ID ?? ""}:${process.env.TWITTER_CLIENT_SECRET ?? ""}`
  ).toString("base64");

  const tokenRes = await fetch("https://api.twitter.com/2/oauth2/token", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      Authorization: `Basic ${basicAuth}`,
    },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: `${appUrl}/api/auth/twitter/callback`,
      code_verifier: codeVerifier,
    }),
  });

  if (!tokenRes.ok) {
    return NextResponse.redirect(`${appUrl}/link?error=twitter_token_exchange_failed`);
  }

  const { access_token } = await tokenRes.json();

  const userRes = await fetch("https://api.twitter.com/2/users/me", {
    headers: { Authorization: `Bearer ${access_token}` },
  });

  if (!userRes.ok) {
    return NextResponse.redirect(`${appUrl}/link?error=twitter_user_fetch_failed`);
  }

  const { data: twitterUser } = await userRes.json();

  const res = NextResponse.redirect(`${appUrl}/link?linked=twitter`);
  res.cookies.set("twitter_handle", `@${twitterUser.username}`, { httpOnly: true, maxAge: 3600, path: "/" });
  res.cookies.delete("twitter_oauth_state");
  res.cookies.delete("twitter_code_verifier");
  return res;
}
