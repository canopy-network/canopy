"use client";
import { useEffect, useLayoutEffect } from "react";
import { usePathname } from "next/navigation";

// useLayoutEffect on the client (reveal in-view sections before paint, no flash),
// useEffect on the server (avoids the "useLayoutEffect does nothing on server" warn).
const useIsoLayout = typeof window !== "undefined" ? useLayoutEffect : useEffect;

// Scan THIS route's .reveal blocks. Mark anything already in view, arm the
// hide-flag, observe the rest. Cleanup disarms so a navigated-away page can't
// leave a later page's sections stuck at opacity:0 (the blank-home-on-nav bug).
function scan() {
  if (typeof window === "undefined") return () => {};
  const html = document.documentElement;
  const reduce =
    window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const els = Array.from(document.querySelectorAll<HTMLElement>(".reveal"));
  if (reduce || typeof IntersectionObserver === "undefined") {
    html.classList.remove("reveal-armed");
    els.forEach((el) => el.classList.add("is-in"));
    return () => {};
  }
  const vh = window.innerHeight || document.documentElement.clientHeight;
  els.forEach((el) => {
    const r = el.getBoundingClientRect();
    if (r.top < vh && r.bottom > 0) el.classList.add("is-in"); // in view -> never hidden
  });
  html.classList.add("reveal-armed");
  const io = new IntersectionObserver(
    (entries) => {
      entries.forEach((e) => {
        if (e.isIntersecting) { e.target.classList.add("is-in"); io.unobserve(e.target); }
      });
    },
    { threshold: 0.12, rootMargin: "0px 0px -8% 0px" }
  );
  els.forEach((el) => { if (!el.classList.contains("is-in")) io.observe(el); });
  return () => { io.disconnect(); html.classList.remove("reveal-armed"); };
}

export function RevealObserver() {
  const pathname = usePathname();
  useIsoLayout(scan, [pathname]); // re-run on every route change
  return null;
}
