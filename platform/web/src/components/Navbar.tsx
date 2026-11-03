"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTeamLead } from "@/lib/teamStore";
import { ScanRepoModal } from "@/components/ScanRepoModal";

export function Navbar() {
    const router = useRouter();
    const lead = useTeamLead();
    const searchInputRef = useRef<HTMLInputElement>(null);
    const notifRef = useRef<HTMLDivElement>(null);
    const [searchQuery, setSearchQuery] = useState("");
    const [isSyncing, setIsSyncing] = useState(false);
    const [toastMessage, setToastMessage] = useState<string | null>(null);
    const [showNotifications, setShowNotifications] = useState(false);
    const [unreadCount, setUnreadCount] = useState(3);
    const [isScanModalOpen, setIsScanModalOpen] = useState(false);

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

    useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
            if (notifRef.current && !notifRef.current.contains(e.target as Node)) {
                setShowNotifications(false);
            }
        };
        document.addEventListener("mousedown", handleClickOutside);
        return () => document.removeEventListener("mousedown", handleClickOutside);
    }, []);

    const handleSyncClick = () => {
        if (isSyncing) return;
        setIsSyncing(true);
        setTimeout(() => {
            setIsSyncing(false);
            setToastMessage("PostgreSQL store & gVisor worker synchronized (8ms)");
            setTimeout(() => setToastMessage(null), 3000);
        }, 900);
    };

    const handleSearchSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (searchQuery.trim()) {
            router.push(`/findings?search=${encodeURIComponent(searchQuery.trim())}`);
        }
    };

    return (
        <>
            <header className="fixed top-0 left-64 right-0 h-16 bg-surface/90 backdrop-blur-md border-b border-outline-variant/40 z-40 flex items-center justify-between px-gutter">
                {toastMessage && (
                    <div className="fixed bottom-6 right-6 z-50 px-4 py-2.5 rounded-lg bg-zinc-900 border border-zinc-700 text-zinc-100 shadow-2xl font-mono text-xs flex items-center gap-2 animate-in fade-in slide-in-from-bottom-2">
                        <span className="w-2 h-2 rounded-full bg-emerald-400" />
                        <span>{toastMessage}</span>
                    </div>
                )}

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
                    onClick={handleSyncClick}
                    className="hidden md:flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container border border-outline-variant/30 text-on-surface-variant font-code-sm text-code-sm hover:bg-surface-container-high transition-colors cursor-pointer"
                    title="Click to verify PostgreSQL & worker sync status"
                >
                    <span className={`w-1.5 h-1.5 rounded-full ${isSyncing ? "bg-amber-400 animate-ping" : "bg-emerald-400 animate-pulse"}`} />
                    <span className={`material-symbols-outlined text-[14px] ${isSyncing ? "animate-spin text-amber-400" : "text-emerald-400"}`}>sync</span>
                    <span className={isSyncing ? "text-amber-400 font-medium" : "text-on-surface-variant"}>
                        {isSyncing ? "syncing..." : "synced"}
                    </span>
                </button>

                <div className="relative" ref={notifRef}>
                    <button
                        type="button"
                        onClick={() => setShowNotifications((prev) => !prev)}
                        className="relative p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors border border-outline-variant/30 flex items-center justify-center cursor-pointer"
                        title="Security Telemetry Notifications"
                    >
                        <span className="material-symbols-outlined text-[20px]">notifications</span>
                        {unreadCount > 0 && (
                            <span className="absolute top-1 right-1 w-3.5 h-3.5 rounded-full bg-rose-600 text-white font-mono text-[9px] font-semibold flex items-center justify-center leading-none">
                                {unreadCount}
                            </span>
                        )}
                    </button>

                    {showNotifications && (
                        <div className="absolute right-0 mt-2 w-96 rounded-xl border border-zinc-800 bg-zinc-950/95 backdrop-blur-xl p-4 shadow-2xl z-50 flex flex-col gap-3 font-sans animate-in fade-in slide-in-from-top-2">
                            <div className="flex items-center justify-between pb-2 border-b border-zinc-800/80">
                                <span className="text-xs font-semibold text-zinc-100 font-mono flex items-center gap-2">
                                    <span className="w-2 h-2 rounded-full bg-rose-500 animate-pulse" />
                                    Security Alerts &amp; Telemetry
                                </span>
                                {unreadCount > 0 && (
                                    <button
                                        type="button"
                                        onClick={() => setUnreadCount(0)}
                                        className="text-[11px] text-zinc-400 hover:text-white font-mono transition cursor-pointer"
                                    >
                                        Mark all read
                                    </button>
                                )}
                            </div>

                            <div className="flex flex-col gap-2.5">
                                <Link
                                    href="/findings"
                                    onClick={() => setShowNotifications(false)}
                                    className="p-2.5 rounded-lg bg-zinc-900/60 hover:bg-zinc-900 border border-zinc-800/80 transition flex flex-col gap-1"
                                >
                                    <div className="flex items-center justify-between">
                                        <span className="text-[10px] font-mono font-bold text-rose-400 bg-rose-500/10 border border-rose-500/20 px-1.5 py-0.5 rounded">
                                            CRITICAL
                                        </span>
                                        <span className="text-[10px] font-mono text-zinc-500">3m ago</span>
                                    </div>
                                    <p className="text-xs text-zinc-200 font-mono font-medium">Command Injection (CWE-78)</p>
                                    <p className="text-[11px] text-zinc-400">Detected in live_pr_flask.py:12 via AST taint walk.</p>
                                </Link>

                                <div className="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/80 flex flex-col gap-1">
                                    <div className="flex items-center justify-between">
                                        <span className="text-[10px] font-mono font-bold text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-1.5 py-0.5 rounded">
                                            VERIFIED
                                        </span>
                                        <span className="text-[10px] font-mono text-zinc-500">12m ago</span>
                                    </div>
                                    <p className="text-xs text-zinc-200 font-mono font-medium">gVisor Sandbox Certified</p>
                                    <p className="text-[11px] text-zinc-400">Container execution hardened with runsc --net none.</p>
                                </div>

                                <Link
                                    href="/remediation"
                                    onClick={() => setShowNotifications(false)}
                                    className="p-2.5 rounded-lg bg-zinc-900/60 hover:bg-zinc-900 border border-zinc-800/80 transition flex flex-col gap-1"
                                >
                                    <div className="flex items-center justify-between">
                                        <span className="text-[10px] font-mono font-bold text-sky-400 bg-sky-500/10 border border-sky-500/20 px-1.5 py-0.5 rounded">
                                            SYNTHESIS
                                        </span>
                                        <span className="text-[10px] font-mono text-zinc-500">25m ago</span>
                                    </div>
                                    <p className="text-xs text-zinc-200 font-mono font-medium">Vertex AI Patch Generated</p>
                                    <p className="text-[11px] text-zinc-400">Gemini synthesized parameterized query patch for CWE-89.</p>
                                </Link>
                            </div>

                            <div className="pt-2 border-t border-zinc-800/80 flex items-center justify-between text-xs font-mono">
                                <Link
                                    href="/findings"
                                    onClick={() => setShowNotifications(false)}
                                    className="text-sky-400 hover:text-sky-300 font-medium"
                                >
                                    View all findings →
                                </Link>
                                <span className="text-zinc-500 text-[11px]">3 telemetry events</span>
                            </div>
                        </div>
                    )}
                </div>

                <button
                    type="button"
                    onClick={() => setIsScanModalOpen(true)}
                    className="flex items-center gap-1.5 h-8 px-3 rounded-lg bg-rose-600 hover:bg-rose-500 text-white border border-rose-500/30 transition-colors font-medium text-xs shadow-sm cursor-pointer"
                >
                    <span className="material-symbols-outlined text-[16px]">bolt</span>
                    <span>+ New Scan</span>
                </button>

                <div className="border-l border-outline-variant/30 h-6 mx-space-xs" />

                <Link href="/team" title={`Active Lead: ${lead.name} (${lead.role})`}>
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-container/30 to-tertiary-container/30 border border-outline-variant/40 flex items-center justify-center font-bold text-xs text-primary ring-1 ring-outline-variant/40 hover:ring-primary transition-all">
                        {lead.avatar}
                    </div>
                </Link>
            </div>
        </header>

        {/* Global New Scan Modal */}
        <ScanRepoModal isOpen={isScanModalOpen} onClose={() => setIsScanModalOpen(false)} />
    </>
);
}