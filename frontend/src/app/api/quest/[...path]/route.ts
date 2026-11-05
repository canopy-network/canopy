import { NextRequest } from "next/server";

const PLUGIN_BASE =
  process.env.QUEST_PLUGIN_URL ||
  "https://arbor.val-a.grad.dev.app.canopynetwork.org/plugin";

export async function GET(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const qs = req.nextUrl.search;
  const upstream = `${PLUGIN_BASE}/v1/query/questxp/${(path ?? []).join("/")}${qs}`;
  try {
    const res = await fetch(upstream, { cache: "no-store" });
    const text = await res.text();
    return new Response(text, {
      status: res.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch {
    return Response.json({ error: "quest service unreachable" }, { status: 502 });
  }
}

export async function POST(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const p = (path ?? []).join("/");
  const target =
    p === "link"
      ? `${PLUGIN_BASE}/v1/link`
      : `${PLUGIN_BASE}/v1/query/questxp/${p}`;
  try {
    const body = await req.text();
    const res = await fetch(target, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });
    const text = await res.text();
    return new Response(text, {
      status: res.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch {
    return Response.json({ error: "quest service unreachable" }, { status: 502 });
  }
}
