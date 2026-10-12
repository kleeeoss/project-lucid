import type { Metadata } from "next";
import "./globals.css";
import { Navbar } from "@/components/Navbar";
import { Sidebar } from "@/components/Sidebar";

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
            <body className="min-h-screen bg-surface font-body-md text-on-surface antialiased selection:bg-primary-container selection:text-on-primary-container">
                <Sidebar />
                <div className="pl-64 flex flex-col min-h-screen">
                    <Navbar />
                    <main className="relative pt-16 w-full flex-1 bg-surface">
                        {children}
                    </main>
                </div>
            </body>
        </html>
    );
}