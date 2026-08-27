import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: { DEFAULT: "#070a0f", 2: "#0c1018" },
        surface: { DEFAULT: "#0f1520", 2: "#141c2e", 3: "#1a2338" },
        line: { DEFAULT: "#1e2a40", 2: "#28374f" },
        ink: { DEFAULT: "#e2eaf8", 2: "#7a90b4", 3: "#374560" },
        up: { DEFAULT: "#00e87a", dim: "rgba(0,232,122,0.06)", glow: "rgba(0,232,122,0.25)" },
        down: { DEFAULT: "#ff4060", dim: "rgba(255,64,96,0.06)" },
        amberx: "#f5a623",
        bluex: "#3b82f6",
        cyanx: "#06b6d4"
      },
      fontFamily: {
        display: ["var(--font-display)", "sans-serif"],
        sans: ["var(--font-sans)", "sans-serif"],
        mono: ["var(--font-mono)", "monospace"]
      },
      borderRadius: { card: "6px" },
      keyframes: {
        fadeUp: { "0%": { opacity: "0", transform: "translateY(8px)" }, "100%": { opacity: "1", transform: "translateY(0)" } },
        pulseDot: { "0%,100%": { opacity: "1" }, "50%": { opacity: "0.3" } }
      },
      animation: {
        fadeUp: "fadeUp 0.4s ease both",
        pulseDot: "pulseDot 2s ease-in-out infinite"
      }
    }
  },
  plugins: []
};
export default config;
