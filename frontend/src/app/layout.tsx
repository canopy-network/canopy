import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "@/components/layout/Providers";
import { Header } from "@/components/layout/Header";
import { TxSubmissionTracker } from "@/components/widgets/TxSubmissionTracker";
import { Space_Grotesk, Manrope } from "next/font/google";
/* arbor-fonts */
const arborDisplay = Space_Grotesk({ subsets: ["latin"], variable: "--font-display", display: "swap" });
const arborBody = Manrope({ subsets: ["latin"], variable: "--font-body", display: "swap" });

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || "http://localhost:3000"),
  title: "ARBOR — Smart Lending Protocol",
  description: "Isolated ARBOR lending markets on Canopy — supply, borrow, and liquidate against live on-chain oracle prices, with real health factors and the protocol bad-debt waterfall.",
  openGraph: { title: "ARBOR — Smart Lending Protocol", description: "Isolated ARBOR lending markets on Canopy — live oracle prices, real health factors, on-chain risk monitor.", images: ["/logo-seal.svg"] },
  icons: { icon: "/logo-seal.svg", apple: "/logo-seal.svg" },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head suppressHydrationWarning>
        {/* eslint-disable-next-line @next/next/no-sync-scripts */}
        {/* eslint-disable-next-line @next/next/no-sync-scripts */}
        <script src="https://cdn.tailwindcss.com"></script>
      </head>
      <body
        className={`${arborDisplay.variable} ${arborBody.variable} min-h-screen bg-[#05070d] text-zinc-100 antialiased`}
        suppressHydrationWarning
      >
        <Providers>
          <Header />

          <main className="mx-auto max-w-6xl px-4 py-6">{children}</main>

          <div className="fixed bottom-4 right-4 z-50 w-full max-w-sm">
            <TxSubmissionTracker />
          </div>
        </Providers>
      </body>
    </html>
  );
}
