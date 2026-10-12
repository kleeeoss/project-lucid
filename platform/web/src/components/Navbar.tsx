"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTeamLead } from "@/lib/teamStore";

export function Navbar() {
    const router = useRouter();
    const lead = useTeamLead();
    const searchInputRef = useRef<HTMLInputElement>(null);
    const [searchQuery, setSearchQuery] = useState("");
    const [isSynced, setIsSynced] = useState(true);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "/" && document.activeElement !== searchInputRef.current) {
                e.preventDefault();
                searchInputRef.current?.focus();
            }
        };
        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, []);

    const handleSearchSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (searchQuery.trim()) {
            router.push(`/findings?search=${encodeURIComponent(searchQuery.trim())}`);
        }
    };

    return (
        <header className="fixed top-0 left-64 right-0 h-16 bg-surface/90 backdrop-blur-md border-b border-outline-variant/40 z-40 flex items-center justify-between px-gutter">
            {/* Left section: Breadcrumbs and Search */}
            <div className="flex items-center gap-space-md flex-1 max-w-2xl">
                <div className="hidden xl:flex items-center gap-space-xs font-code-sm text-code-sm text-on-surface-variant">
                    <span className="material-symbols-outlined text-[16px]">corporate_fare</span>
                    <span>lucid-security-org</span>
                    <span className="text-outline">/</span>
                    <span className="text-on-surface font-medium">main</span>
                </div>

                <form onSubmit={handleSearchSubmit} className="relative flex-1">
                    <span className="material-symbols-outlined absolute left-2.5 top-1/2 -translate-y-1/2 text-[18px] text-on-surface-variant pointer-events-none">
                        search
                    </span>
                    <input
                        ref={searchInputRef}
                        type="text"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        placeholder="Search repositories, PRs, findings... (Press /)"
                        className="w-full h-8 pl-8 pr-3 rounded-lg bg-surface-container-low border border-outline-variant/40 text-on-surface placeholder:text-on-surface-variant/60 font-body-sm text-body-sm focus:outline-none focus:border-primary-container focus:ring-1 focus:ring-primary-container transition-all"
                    />
                </form>
            </div>

            {/* Right section: Sync status, Notifications, Primary Action, Avatar */}
            <div className="flex items-center gap-space-sm">
                <button
                    type="button"
                    onClick={() => setIsSynced((prev) => !prev)}
                    className="hidden md:flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container border border-outline-variant/30 text-on-surface-variant font-code-sm text-code-sm hover:bg-surface-container-high transition-colors"
                    title="Click to check sync status"
                >
                    <span className={`w-1.5 h-1.5 rounded-full ${isSynced ? "bg-secondary animate-pulse" : "bg-warning"}`} />
                    <span className="material-symbols-outlined text-[14px]">sync</span>
                    <span>{isSynced ? "synced" : "syncing..."}</span>
                </button>

                <Link
                    href="/findings"
                    className="relative p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors border border-outline-variant/30 flex items-center justify-center"
                    title="Notifications"
                >
                    <span className="material-symbols-outlined text-[20px]">notifications</span>
                    <span className="absolute top-1 right-1 w-3.5 h-3.5 rounded-full bg-error text-on-error font-code-sm text-[9px] font-semibold flex items-center justify-center leading-none">
                        3
                    </span>
                </Link>

                <Link
                    href="/scans"
                    className="flex items-center gap-1.5 h-8 px-space-md rounded-lg bg-secondary-container hover:bg-secondary text-on-secondary-container hover:text-on-secondary border border-secondary transition-colors font-label-md text-label-md font-semibold"
                >
                    <span className="material-symbols-outlined text-[16px]">bolt</span>
                    <span>Scan Repository</span>
                </Link>

                <div className="border-l border-outline-variant/30 h-6 mx-space-xs" />

                <Link href="/team" title={`Active Lead: ${lead.name} (${lead.role})`}>
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-container/30 to-tertiary-container/30 border border-outline-variant/40 flex items-center justify-center font-bold text-xs text-primary ring-1 ring-outline-variant/40 hover:ring-primary transition-all">
                        {lead.avatar}
                    </div>
                </Link>
            </div>
        </header>
    );
}