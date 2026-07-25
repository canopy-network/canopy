import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "@/components/layout/Providers";
import { Header } from "@/components/layout/Header";
import { TxSubmissionTracker } from "@/components/widgets/TxSubmissionTracker";

export const metadata: Metadata = {
  title: "ARBOR | Canopy",
  description: "ARBOR lending protocol frontend for Canopy",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head suppressHydrationWarning>
        <script src="https://cdn.tailwindcss.com"></script>
      </head>
      <body
        className="min-h-screen bg-[#05070d] text-zinc-100 antialiased"
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
