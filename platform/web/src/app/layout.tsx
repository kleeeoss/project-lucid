import type { Metadata } from "next";
import "./globals.css";
import { Navbar } from "@/components/Navbar";

export const metadata: Metadata = {
    title: "Lucid-CI — DevSecOps Intelligence Gateway",
    description: "Automated Pull Request Security Analysis & SLM Patch Remediation",
};

export default function RootLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <html lang="en" className="dark">
            <body className="min-h-screen bg-[#090d16] text-slate-100 antialiased">
                <Navbar />
                <main className="mx-auto max-w-7xl px-6 py-8">{children}</main>
            </body>
        </html>
    );
}