"use client";

export function EmptyState({
  message,
  sub,
}: {
  message: string;
  sub?: string;
}) {
  return (
    <div className="rounded-2xl glass backdrop-blur p-8 text-center">
      <p className="text-sm text-zinc-400">{message}</p>
      {sub && <p className="mt-2 text-xs text-zinc-600">{sub}</p>}
    </div>
  );
}
