import type { Metadata } from "next";
import "./globals.css";
import { AppHeader } from "@/components/app-header";

export const metadata: Metadata = {
  title: "NEO Mini · Quản lý đơn hàng",
  description: "Bảng điều khiển đơn hàng và tồn kho NEO Mini",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="vi">
      <body>
        <AppHeader />
        <main className="mx-auto min-h-[calc(100vh-76px)] w-full max-w-6xl px-5 pb-16 pt-10 sm:px-8">
          {children}
        </main>
      </body>
    </html>
  );
}
