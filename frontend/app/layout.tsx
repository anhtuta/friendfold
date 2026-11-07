import type { Metadata } from "next";
import { Inter } from "next/font/google";

import { AuthProvider } from "@/components/AuthProvider";
import { SiteHeader } from "@/components/SiteHeader";

import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Friendfold",
  description: "Into the fold of friends.",
};

// Next.js App Router requires a default export from app/layout.tsx.
export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className="bg-background text-foreground">
      <body
        className={`${inter.className} min-h-screen bg-background text-foreground antialiased`}
      >
        <AuthProvider>
          <SiteHeader />
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}
