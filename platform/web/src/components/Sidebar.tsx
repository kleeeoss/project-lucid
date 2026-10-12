"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTeamLead } from "@/lib/teamStore";

interface NavItem {
    href: string;
    label: string;
    icon: string;
}

const securityCoreLinks: NavItem[] = [
    { href: "/", label: "Overview", icon: "grid_view" },
    { href: "/repositories", label: "Repositories", icon: "source" },
    { href: "/pull-requests", label: "Pull Requests", icon: "merge_type" },
    { href: "/scans", label: "Scans", icon: "radar" },
    { href: "/findings", label: "Findings", icon: "bug_report" },
    { href: "/remediation", label: "Remediation", icon: "auto_fix_high" },
    { href: "/analytics", label: "Analytics", icon: "query_stats" },
];

const workspaceLinks: NavItem[] = [
    { href: "/team", label: "Team", icon: "group" },
    { href: "/integrations", label: "Integrations", icon: "extension" },
    { href: "/settings", label: "Settings", icon: "tune" },
];

export function Sidebar() {
    const pathname = usePathname();
    const lead = useTeamLead();

    function isLinkActive(href: string) {
        if (href === "/") return pathname === "/";
        return pathname.startsWith(href);
    }

    return (
        <aside className="fixed left-0 top-0 h-full w-64 bg-surface-container-low border-r border-outline-variant/40 z-50 flex flex-col justify-between select-none">
            <div className="flex flex-col min-h-0 flex-1">
                {/* Brand / Logo header */}
                <div className="h-16 px-gutter flex items-center justify-between border-b border-outline-variant/30">
                    <Link href="/" className="flex items-center gap-space-sm hover:opacity-95 transition-opacity">
                        <div className="h-7 w-7 rounded-md bg-surface-container-high border border-outline-variant/50 flex items-center justify-center text-primary shadow-sm">
                            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24">
                                <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 2.18l7 3.12v4.7c0 4.54-3.14 8.78-7 9.88-3.86-1.1-7-5.34-7-9.88V6.3l7-3.12zm-1 5.82v6h2v-6h-2z" />
                            </svg>
                        </div>
                        <span className="font-headline-md text-headline-md tracking-tight text-on-surface font-semibold">
                            Lucid
                        </span>
                        <span className="px-space-xs py-0.5 rounded-lg bg-surface-container-highest text-on-surface-variant font-code-sm text-code-sm border border-outline-variant/40">
                            v2.4
                        </span>
                    </Link>
                </div>

                {/* Navigation items */}
                <div className="flex-1 overflow-y-auto px-space-sm py-space-md">
                    <div className="px-space-sm pb-space-xs text-on-surface-variant font-code-sm text-code-sm tracking-wider uppercase opacity-60">
                        Security Core
                    </div>
                    <nav className="flex flex-col gap-0.5">
                        {securityCoreLinks.map((item) => {
                            const active = isLinkActive(item.href);
                            return (
                                <Link
                                    key={item.href}
                                    href={item.href}
                                    className={`flex items-center gap-space-sm px-space-sm py-1.5 rounded-lg transition-all ${
                                        active
                                            ? "bg-surface-container-high text-primary font-medium border border-outline-variant/60 shadow-sm"
                                            : "text-on-surface-variant hover:bg-surface-container hover:text-on-surface font-body-md text-body-md"
                                    }`}
                                >
                                    <span className="material-symbols-outlined text-[18px]">
                                        {item.icon}
                                    </span>
                                    <span>{item.label}</span>
                                </Link>
                            );
                        })}
                    </nav>

                    <div className="my-space-md border-t border-outline-variant/30 mx-space-xs" />

                    <div className="px-space-sm pb-space-xs text-on-surface-variant font-code-sm text-code-sm tracking-wider uppercase opacity-60">
                        Workspace
                    </div>
                    <nav className="flex flex-col gap-0.5">
                        {workspaceLinks.map((item) => {
                            const active = isLinkActive(item.href);
                            return (
                                <Link
                                    key={item.href}
                                    href={item.href}
                                    className={`flex items-center gap-space-sm px-space-sm py-1.5 rounded-lg transition-all ${
                                        active
                                            ? "bg-surface-container-high text-primary font-medium border border-outline-variant/60 shadow-sm"
                                            : "text-on-surface-variant hover:bg-surface-container hover:text-on-surface font-body-md text-body-md"
                                    }`}
                                >
                                    <span className="material-symbols-outlined text-[18px]">
                                        {item.icon}
                                    </span>
                                    <span>{item.label}</span>
                                </Link>
                            );
                        })}
                    </nav>
                </div>
            </div>

            {/* Bottom User & GitHub integration profile */}
            <div className="p-space-sm border-t border-outline-variant/30 bg-surface-container-lowest/50">
                <Link
                    href="/team"
                    className="flex items-center gap-space-sm p-space-xs rounded-lg hover:bg-surface-container transition-colors cursor-pointer group"
                    title="Manage team members & lead"
                >
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-container/30 to-tertiary-container/30 border border-outline-variant/40 flex items-center justify-center font-bold text-xs text-primary shrink-0 group-hover:border-primary/50 transition-colors">
                        {lead.avatar}
                    </div>
                    <div className="flex-1 min-w-0">
                        <div className="font-label-md text-label-md text-on-surface truncate font-semibold">
                            {lead.name}
                        </div>
                        <div className="font-code-sm text-code-sm text-on-surface-variant truncate">
                            {lead.role}
                        </div>
                    </div>
                </Link>
                <div className="mt-space-xs flex items-center gap-1.5 px-space-xs py-1 rounded bg-surface-container-low border border-outline-variant/20 font-code-sm text-code-sm text-on-surface-variant">
                    <span className="w-2 h-2 rounded-full bg-secondary shrink-0" />
                    <span className="truncate font-label-sm text-label-sm">GitHub: {lead.github || "@workspace-lead"}</span>
                </div>
            </div>
        </aside>
    );
}
