import type {
  InputHTMLAttributes,
  SelectHTMLAttributes,
  ButtonHTMLAttributes,
  ReactNode,
} from "react";

export function Field({
  label,
  error,
  hint,
  children,
}: {
  label: string;
  error?: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium text-zinc-400">{label}</span>
      {children}
      {hint && !error && (
        <span className="block text-[11px] text-zinc-600">{hint}</span>
      )}
      {error && (
        <span className="block text-[11px] text-rose-400">{error}</span>
      )}
    </label>
  );
}

const fieldBase =
  "w-full rounded-xl border bg-white/[0.02] px-3.5 py-2.5 text-sm text-zinc-100 outline-none transition placeholder:text-zinc-600 focus:ring-2";

function stateClasses(invalid?: boolean) {
  return invalid
    ? "border-rose-500/50 focus:border-rose-400/60 focus:ring-rose-500/20"
    : "border-white/10 focus:border-indigo-400/60 focus:ring-indigo-500/20";
}

export function TextInput({
  invalid,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & { invalid?: boolean }) {
  return (
    <input
      {...rest}
      className={`${fieldBase} ${stateClasses(invalid)} ${rest.className ?? ""}`}
    />
  );
}

export function SelectInput({
  invalid,
  children,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement> & { invalid?: boolean }) {
  return (
    <div className="relative">
      <select
        {...rest}
        className={`${fieldBase} appearance-none pr-9 ${stateClasses(invalid)} ${rest.className ?? ""}`}
      >
        {children}
      </select>
      <svg
        className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M5.23 7.21a.75.75 0 011.06.02L10 11.17l3.71-3.94a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
          clipRule="evenodd"
        />
      </svg>
    </div>
  );
}

export function SubmitButton({
  busy,
  children,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { busy?: boolean }) {
  return (
    <button
      {...rest}
      disabled={rest.disabled || busy}
      className={`inline-flex w-full items-center justify-center gap-2 rounded-xl btn-brand px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400 disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none ${rest.className ?? ""}`}
    >
      {busy && (
        <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-90"
            fill="currentColor"
            d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
          />
        </svg>
      )}
      {children}
    </button>
  );
}

export function ErrorText({ children }: { children: ReactNode }) {
  return <p className="text-sm text-rose-400">{children}</p>;
}
