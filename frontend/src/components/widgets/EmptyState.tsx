"use client";

export function EmptyState({
  message,
  sub,
}: {
  message: string;
  sub?: string;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-8 text-center">
      <p className="text-sm text-zinc-400">{message}</p>
      {sub && <p className="mt-2 text-xs text-zinc-600">{sub}</p>}
    </div>
  );
}
