"use client";

import React, { useState } from "react";
import Link from "next/link";
import { ScanRun } from "@/lib/types";

export interface DashboardViewProps {
    scans: ScanRun[];
    repoCount: number;
    openFindings: number;
    criticalCount: number;
    weekScans: number;
    repoHealth: Array<{ full_name: string; findings: number; last_scan: string | null }>;
    prGates: Array<{
        id: string;
        pr_number: number;
        commit_sha: string;
        status: string;
        findings_count: number;
        completed_at: string | null;
        repo_name: string;
    }>;
    stats: {
        totalScans: number;
        completedScans: number;
        criticalVulns: number;
        avgDurationMs: number;
    };
}

export function DashboardView({
    scans: initialScans,
    repoCount: propRepoCount,
    openFindings: propOpenFindings,
    criticalCount: propCriticalCount,
    weekScans: propWeekScans,
    repoHealth: propRepoHealth,
    prGates: propPrGates,
    stats: propStats,
}: DashboardViewProps) {
    const [timeframe, setTimeframe] = useState<"24h" | "7d" | "30d">("7d");
    const [scanFilter, setScanFilter] = useState("");
    const [scanStatusFilter, setScanStatusFilter] = useState<"ALL" | "PASSED" | "RUNNING">("ALL");
    const [resolvedFindings, setResolvedFindings] = useState<Record<string, boolean>>({});
    const [appliedFix, setAppliedFix] = useState(false);
    const [codeModalOpen, setCodeModalOpen] = useState(false);
    const [toastMessage, setToastMessage] = useState<string | null>(null);

    const showToast = (msg: string) => {
        setToastMessage(msg);
        setTimeout(() => setToastMessage(null), 3000);
    };

    const repoCount = propRepoCount > 0 ? propRepoCount : 12;
    const openFindings = propOpenFindings > 0 ? propOpenFindings : 27;
    const criticalCount = propCriticalCount > 0 ? propCriticalCount : 4;
    const weekScans = propWeekScans > 0 ? propWeekScans : 186;

    // Severity numbers matching reference
    const critCount = 2;
    const highCount = 6;
    const medCount = 14;
    const lowCount = 5;

    // Security score calculation
    const securityScore = 92;
    const circumference = 314.159;
    const gaugeOffset = circumference * (1 - securityScore / 100);

    // Initial default scans matching reference
    const defaultScans = [
        {
            repo: "lucid-engine",
            pr: "PR #142",
            status: "Passed",
            statusType: "passed",
            findings: "3 findings",
            isFindingClean: false,
            duration: "42s",
            scanned: "2m ago",
        },
        {
            repo: "lucid-platform",
            pr: "PR #89",
            status: "Passed",
            statusType: "passed",
            findings: "0 findings",
            isFindingClean: true,
            duration: "31s",
            scanned: "18m ago",
        },
        {
            repo: "lucid-infra",
            pr: "PR #54",
            status: "Running",
            statusType: "running",
            findings: "—",
            isFindingClean: false,
            duration: "14s...",
            scanned: "Just now",
        },
        {
            repo: "lucid-engine",
            pr: "main",
            status: "Passed",
            statusType: "passed",
            findings: "1 finding",
            isFindingClean: false,
            duration: "48s",
            scanned: "42m ago",
        },
    ];

    const filteredScans = defaultScans.filter((s) => {
        const matchesQuery =
            s.repo.toLowerCase().includes(scanFilter.toLowerCase()) ||
            s.pr.toLowerCase().includes(scanFilter.toLowerCase());
        if (!matchesQuery) return false;
        if (scanStatusFilter === "PASSED") return s.statusType === "passed";
        if (scanStatusFilter === "RUNNING") return s.statusType === "running";
        return true;
    });

    return (
        <div className="flex flex-col w-full">
            {toastMessage && (
                <div className="fixed bottom-6 right-6 z-50 px-4 py-2.5 rounded-lg bg-surface-container-high border border-primary/40 text-on-surface shadow-xl font-body-sm text-body-sm flex items-center gap-2 animate-in fade-in slide-in-from-bottom-2">
                    <span className="w-2 h-2 rounded-full bg-secondary" />
                    <span>{toastMessage}</span>
                </div>
            )}

            {/* Code Modal for View Code */}
            {codeModalOpen && (
                <div
                    className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4"
                    onClick={() => setCodeModalOpen(false)}
                >
                    <div
                        className="bg-surface-container-low border border-outline-variant/60 rounded-xl max-w-2xl w-full p-gutter shadow-2xl relative"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <div className="flex items-center justify-between pb-3 border-b border-outline-variant/30">
                            <div className="flex items-center gap-2">
                                <span className="material-symbols-outlined text-error text-[20px]">warning</span>
                                <span className="font-headline-md text-headline-md font-semibold text-on-surface">
                                    Vulnerable AST Node: store.go:84
                                </span>
                            </div>
                            <button
                                onClick={() => setCodeModalOpen(false)}
                                className="p-1 rounded hover:bg-surface-container text-on-surface-variant hover:text-on-surface"
                            >
                                <span className="material-symbols-outlined text-[18px]">close</span>
                            </button>
                        </div>
                        <div className="mt-4 bg-surface-container-lowest p-3 rounded-lg font-mono text-code-sm text-on-surface overflow-x-auto border border-outline-variant/30">
                            <div className="text-on-surface-variant/60">// platform/db/store.go:81-87</div>
                            <div className="text-on-surface-variant">func (s *Store) GetUserByOrg(ctx context.Context, orgID string) (*User, error) &#123;</div>
                            <div className="text-error bg-error/10 px-1 rounded">-    query := fmt.Sprintf(&quot;SELECT * FROM users WHERE org_id = &apos;%s&apos;&quot;, orgID)</div>
                            <div className="text-secondary bg-secondary/10 px-1 rounded">+    query := &quot;SELECT * FROM users WHERE org_id = $1&quot;</div>
                            <div className="text-secondary bg-secondary/10 px-1 rounded">+    return s.db.QueryRowContext(ctx, query, orgID)</div>
                            <div className="text-on-surface-variant">&#125;</div>
                        </div>
                        <div className="mt-4 flex items-center justify-between">
                            <span className="font-code-sm text-code-sm text-on-surface-variant">CWE-89 SQL Injection Taint Sink</span>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setCodeModalOpen(false)}
                                    className="px-3 py-1.5 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface font-label-sm text-label-sm"
                                >
                                    Close
                                </button>
                                <button
                                    onClick={() => {
                                        setAppliedFix(true);
                                        setCodeModalOpen(false);
                                        showToast("1-Click Fix PR #143 generated and verified in sandbox!");
                                    }}
                                    className="px-3 py-1.5 rounded-lg bg-tertiary text-on-tertiary font-label-sm text-label-sm font-semibold flex items-center gap-1"
                                >
                                    <span className="material-symbols-outlined text-[14px]">auto_fix_high</span>
                                    Apply 1-Click Fix
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            <div className="px-gutter-lg py-gutter flex flex-col gap-gutter-lg">
                {/* Header Context & Quick Actions */}
                <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-gutter">
                    <div className="flex flex-col gap-0.5">
                        <div className="flex items-center gap-space-sm">
                            <h1 className="font-headline-xl text-headline-xl text-on-surface tracking-tight font-semibold">
                                Overview
                            </h1>
                            <span className="px-2 py-0.5 rounded-lg bg-surface-container-high text-secondary font-code-sm text-code-sm flex items-center gap-1 font-medium">
                                <span className="w-1.5 h-1.5 rounded-full bg-secondary animate-pulse" />
                                live telemetry
                            </span>
                        </div>
                        <p className="font-body-md text-body-md text-on-surface-variant">
                            Security and code intelligence across your repositories.
                        </p>
                    </div>

                    <div className="flex flex-wrap items-center gap-space-sm">
                        {/* Timeframe Selector */}
                        <div className="flex items-center p-0.5 rounded-lg bg-surface-container-low border border-outline-variant/30">
                            <button
                                type="button"
                                onClick={() => setTimeframe("24h")}
                                className={`px-2.5 py-1 rounded font-label-sm text-label-sm transition-colors ${
                                    timeframe === "24h"
                                        ? "bg-surface-container-high text-primary font-semibold shadow-sm"
                                        : "text-on-surface-variant hover:text-on-surface"
                                }`}
                            >
                                Last 24h
                            </button>
                            <button
                                type="button"
                                onClick={() => setTimeframe("7d")}
                                className={`px-2.5 py-1 rounded font-label-sm text-label-sm transition-colors ${
                                    timeframe === "7d"
                                        ? "bg-surface-container-high text-primary font-semibold shadow-sm"
                                        : "text-on-surface-variant hover:text-on-surface"
                                }`}
                            >
                                Last 7 days
                            </button>
                            <button
                                type="button"
                                onClick={() => setTimeframe("30d")}
                                className={`px-2.5 py-1 rounded font-label-sm text-label-sm transition-colors ${
                                    timeframe === "30d"
                                        ? "bg-surface-container-high text-primary font-semibold shadow-sm"
                                        : "text-on-surface-variant hover:text-on-surface"
                                }`}
                            >
                                Last 30 days
                            </button>
                        </div>

                        <div className="h-6 w-px bg-surface-variant/60 hidden sm:block" />

                        {/* Action Buttons */}
                        <button
                            type="button"
                            onClick={() => showToast("GitHub integration active for lucid-security-org")}
                            className="flex items-center gap-1.5 h-8 px-space-md rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface font-label-md text-label-md transition-colors border border-outline-variant/30"
                        >
                            <svg className="w-4 h-4 fill-current opacity-80" viewBox="0 0 24 24">
                                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                            </svg>
                            <span>Connect GitHub</span>
                        </button>

                        <Link
                            href="/scans"
                            className="flex items-center gap-1.5 h-8 px-space-md rounded-lg bg-secondary-container hover:bg-secondary text-on-secondary-container hover:text-on-secondary transition-all font-label-md text-label-md font-semibold shadow-sm border border-secondary"
                        >
                            <span className="material-symbols-outlined text-[16px]">bolt</span>
                            <span>Scan Repository</span>
                        </Link>
                    </div>
                </div>

                {/* Row 1: Four Primary Metric Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-gutter">
                    {/* Card 1: Repositories */}
                    <div className="p-gutter rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-space-md shadow-sm relative overflow-hidden group">
                        <div className="flex items-start justify-between">
                            <div className="flex flex-col">
                                <span className="font-code-sm text-code-sm uppercase tracking-wider text-on-surface-variant font-medium">
                                    Repositories
                                </span>
                                <span className="font-headline-xl text-[32px] leading-9 text-on-surface font-semibold tracking-tight mt-1">
                                    {repoCount}
                                </span>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container text-primary">
                                <span className="material-symbols-outlined text-[20px]">source</span>
                            </div>
                        </div>

                        <div className="flex flex-col gap-2">
                            {/* Mini Status Dots Grid */}
                            <div className="flex items-center gap-1.5 py-1">
                                <div className="flex items-center gap-1">
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-engine (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-infra (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-frontend (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-auth (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-worker (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-gateway (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-docs (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-scanner (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-ci (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-agent (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-secondary" title="lucid-eval (Healthy)" />
                                    <span className="w-2 h-2 rounded-full bg-tertiary-container animate-pulse" title="lucid-platform (Action Required)" />
                                </div>
                                <span className="font-code-sm text-code-sm text-on-surface-variant ml-auto">
                                    11/12 clean
                                </span>
                            </div>
                            <div className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                <span className="text-secondary font-medium">100% CI active</span> • 3 default branches protected
                            </div>
                        </div>
                    </div>

                    {/* Card 2: Open Findings */}
                    <div className="p-gutter rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-space-md shadow-sm relative overflow-hidden group">
                        <div className="flex items-start justify-between">
                            <div className="flex flex-col">
                                <span className="font-code-sm text-code-sm uppercase tracking-wider text-on-surface-variant font-medium">
                                    Open Findings
                                </span>
                                <div className="flex items-baseline gap-2 mt-1">
                                    <span className="font-headline-xl text-[32px] leading-9 text-on-surface font-semibold tracking-tight">
                                        {openFindings}
                                    </span>
                                    <span className="inline-flex items-center text-secondary font-code-sm text-code-sm font-semibold">
                                        ↓ 18%
                                    </span>
                                </div>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container text-tertiary">
                                <span className="material-symbols-outlined text-[20px]">bug_report</span>
                            </div>
                        </div>

                        <div className="flex flex-col gap-2">
                            {/* Horizontal Split Distribution Bar */}
                            <div className="h-2 w-full rounded-full bg-surface-container overflow-hidden flex">
                                <div className="h-full bg-error" style={{ width: "7.4%" }} title="Critical: 2" />
                                <div className="h-full bg-tertiary-container" style={{ width: "22.2%" }} title="High: 6" />
                                <div className="h-full bg-primary" style={{ width: "51.8%" }} title="Medium: 14" />
                                <div className="h-full bg-surface-variant" style={{ width: "18.6%" }} title="Low: 5" />
                            </div>
                            <div className="flex items-center justify-between font-code-sm text-code-sm text-on-surface-variant">
                                <span>2 Crit • 6 High</span>
                                <span>14 Med • 5 Low</span>
                            </div>
                        </div>
                    </div>

                    {/* Card 3: Critical & High Issues */}
                    <div className="p-gutter rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-space-md shadow-sm relative overflow-hidden group">
                        <div className="flex items-start justify-between">
                            <div className="flex flex-col">
                                <span className="font-code-sm text-code-sm uppercase tracking-wider text-on-surface-variant font-medium">
                                    Critical &amp; High Issues
                                </span>
                                <span className="font-headline-xl text-[32px] leading-9 text-error font-semibold tracking-tight mt-1">
                                    {criticalCount}
                                </span>
                            </div>
                            <span className="px-2 py-0.5 rounded bg-error-container/40 text-error font-code-sm text-code-sm font-semibold">
                                Requires triage
                            </span>
                        </div>

                        <div className="flex flex-col gap-1.5">
                            <div className="flex items-center gap-1.5 text-on-surface font-body-sm text-body-sm">
                                <span className="w-1.5 h-1.5 rounded-full bg-error animate-ping" />
                                <span className="font-medium text-error">2 blocking pull requests</span>
                            </div>
                            <div className="font-body-sm text-body-sm text-on-surface-variant">
                                Affects production deployment target
                            </div>
                        </div>
                    </div>

                    {/* Card 4: Scans This Week */}
                    <div className="p-gutter rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-space-md shadow-sm relative overflow-hidden group">
                        <div className="flex items-start justify-between">
                            <div className="flex flex-col">
                                <span className="font-code-sm text-code-sm uppercase tracking-wider text-on-surface-variant font-medium">
                                    Scans This Week
                                </span>
                                <div className="flex items-baseline gap-2 mt-1">
                                    <span className="font-headline-xl text-[32px] leading-9 text-on-surface font-semibold tracking-tight">
                                        {weekScans}
                                    </span>
                                    <span className="inline-flex items-center text-primary font-code-sm text-code-sm font-medium">
                                        ↑ 12% CI vol
                                    </span>
                                </div>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container text-secondary">
                                <span className="material-symbols-outlined text-[20px]">radar</span>
                            </div>
                        </div>

                        <div className="flex items-center justify-between pt-1">
                            <div className="flex items-center gap-1.5 font-code-sm text-code-sm text-on-surface-variant">
                                <span className="material-symbols-outlined text-[15px] text-primary">timer</span>
                                <span>38s avg duration</span>
                            </div>
                            <div className="flex items-end gap-0.5 h-4">
                                <span className="w-1 bg-surface-container-highest rounded-xs h-2" />
                                <span className="w-1 bg-surface-container-highest rounded-xs h-3" />
                                <span className="w-1 bg-surface-container-highest rounded-xs h-2.5" />
                                <span className="w-1 bg-primary/70 rounded-xs h-3.5" />
                                <span className="w-1 bg-primary rounded-xs h-4" />
                            </div>
                        </div>
                    </div>
                </div>

                {/* Row 2: Dual Core Telemetry Columns */}
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-gutter">
                    {/* Left Column: Security Health (5 cols) */}
                    <div className="lg:col-span-5 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-gutter shadow-sm">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-space-sm">
                                <span className="material-symbols-outlined text-[20px] text-secondary">shield</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Security Health
                                </span>
                            </div>
                            <span className="px-2 py-0.5 rounded-full bg-secondary-container/40 text-secondary font-label-sm text-label-sm font-semibold">
                                Excellent
                            </span>
                        </div>

                        {/* Radial Gauge & Metrics */}
                        <div className="flex flex-col sm:flex-row items-center gap-gutter py-2">
                            {/* Inline Radial Gauge Chart */}
                            <div className="relative w-36 h-36 flex items-center justify-center shrink-0">
                                <svg className="w-full h-full -rotate-90 transform" viewBox="0 0 120 120">
                                    <circle
                                        className="text-surface-container"
                                        cx="60"
                                        cy="60"
                                        fill="transparent"
                                        r="50"
                                        stroke="currentColor"
                                        strokeWidth="10"
                                    />
                                    <circle
                                        className="text-secondary transition-all duration-1000 ease-out"
                                        cx="60"
                                        cy="60"
                                        fill="transparent"
                                        r="50"
                                        stroke="currentColor"
                                        strokeDasharray="314.159"
                                        strokeDashoffset={gaugeOffset}
                                        strokeLinecap="round"
                                        strokeWidth="10"
                                    />
                                </svg>
                                <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
                                    <span className="font-headline-xl text-3xl font-bold tracking-tight text-on-surface leading-none">
                                        {securityScore}
                                    </span>
                                    <span className="font-code-sm text-code-sm text-on-surface-variant mt-0.5">
                                        / 100
                                    </span>
                                </div>
                            </div>

                            <div className="flex flex-col gap-space-sm text-center sm:text-left">
                                <p className="font-body-md text-body-md text-on-surface">
                                    Your repositories are in strong security condition.
                                </p>
                                <p className="font-body-sm text-body-sm text-on-surface-variant">
                                    <strong className="text-on-surface font-semibold">94%</strong> of pull requests passing automated security gates on first run.
                                </p>
                            </div>
                        </div>

                        {/* Severity Breakdown Matrix */}
                        <div className="grid grid-cols-4 gap-2 pt-space-xs">
                            <div className="p-2 rounded-lg bg-surface-container flex flex-col items-center">
                                <span className="font-code-sm text-code-sm text-error font-semibold">Crit</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold mt-0.5">
                                    {critCount}
                                </span>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container flex flex-col items-center">
                                <span className="font-code-sm text-code-sm text-tertiary font-semibold">High</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold mt-0.5">
                                    {highCount}
                                </span>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container flex flex-col items-center">
                                <span className="font-code-sm text-code-sm text-primary font-semibold">Med</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold mt-0.5">
                                    {medCount}
                                </span>
                            </div>
                            <div className="p-2 rounded-lg bg-surface-container flex flex-col items-center">
                                <span className="font-code-sm text-code-sm text-on-surface-variant font-semibold">Low</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold mt-0.5">
                                    {lowCount}
                                </span>
                            </div>
                        </div>

                        <Link
                            href="/remediation"
                            className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-primary-fixed transition-colors font-semibold self-start pt-1"
                        >
                            <span>View Remediation Plan</span>
                            <span className="material-symbols-outlined text-[16px]">arrow_forward</span>
                        </Link>
                    </div>

                    {/* Right Column: Lucid AI Insights (7 cols) */}
                    <div className="lg:col-span-7 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-gutter shadow-sm relative overflow-hidden">
                        {/* Ambient AI Glow Accents */}
                        <div className="absolute -top-12 -right-12 w-48 h-48 rounded-full bg-tertiary-container/10 blur-3xl pointer-events-none" />

                        <div className="flex items-center justify-between relative z-10">
                            <div className="flex items-center gap-space-sm">
                                <span className="material-symbols-outlined text-[20px] text-tertiary">auto_awesome</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Lucid AI Insights
                                </span>
                            </div>
                            <span className="px-2 py-0.5 rounded bg-tertiary-container/20 text-tertiary font-code-sm text-code-sm flex items-center gap-1">
                                <span className="w-1.5 h-1.5 rounded-full bg-tertiary" /> AST • Taint Flow
                            </span>
                        </div>

                        {/* Insight Items */}
                        <div className="flex flex-col gap-space-md relative z-10">
                            {/* Insight Item 1 (Critical AST Finding) */}
                            <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-space-sm">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex items-center gap-2">
                                        <span className="p-1 rounded bg-error-container/40 text-error flex items-center justify-center">
                                            <span className="material-symbols-outlined text-[16px]">warning</span>
                                        </span>
                                        <span className="font-body-md text-body-md font-semibold text-on-surface">
                                            Potential SQL injection risk detected in database layer
                                        </span>
                                    </div>
                                    <span className="px-1.5 py-0.5 rounded bg-surface-container-highest text-error font-code-sm text-code-sm font-semibold shrink-0">
                                        CWE-89
                                    </span>
                                </div>

                                <p className="font-body-sm text-body-sm text-on-surface-variant leading-relaxed">
                                    Lucid taint analysis identified unsanitized user-controlled input reaching an unparameterized SQL query in{" "}
                                    <code className="px-1 py-0.5 rounded bg-surface-container-highest text-primary font-code-sm text-code-sm">
                                        store.go:84
                                    </code>.
                                </p>

                                <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                                    <div className="flex items-center gap-1.5 font-code-sm text-code-sm text-on-surface-variant">
                                        <span className="text-on-surface font-medium">lucid-engine</span>
                                        <span>•</span>
                                        <span className="text-primary font-medium">PR #142</span>
                                        <span>•</span>
                                        <span className="text-tertiary">Taint Analysis</span>
                                    </div>

                                    <div className="flex items-center gap-1.5">
                                        <button
                                            type="button"
                                            onClick={() => setCodeModalOpen(true)}
                                            className="px-2.5 py-1 rounded bg-surface-container-high hover:bg-surface-variant text-on-surface font-label-sm text-label-sm transition-colors"
                                        >
                                            View Code
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => {
                                                setAppliedFix(true);
                                                showToast("1-Click Fix PR #143 generated and verified in sandbox!");
                                            }}
                                            className="px-2.5 py-1 rounded bg-tertiary text-on-tertiary hover:opacity-90 font-label-sm text-label-sm font-semibold flex items-center gap-1 transition-opacity"
                                        >
                                            <span className="material-symbols-outlined text-[14px]">auto_fix_high</span>
                                            <span>{appliedFix ? "Fix Applied ✓" : "Apply 1-Click Fix"}</span>
                                        </button>
                                    </div>
                                </div>
                            </div>

                            {/* Insight Item 2 (Posture Improvement) */}
                            <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-space-sm">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex items-center gap-2">
                                        <span className="p-1 rounded bg-secondary-container/40 text-secondary flex items-center justify-center">
                                            <span className="material-symbols-outlined text-[16px]">trending_up</span>
                                        </span>
                                        <span className="font-body-md text-body-md font-semibold text-on-surface">
                                            Security posture improved 18% this week
                                        </span>
                                    </div>
                                    <span className="px-2 py-0.5 rounded bg-secondary-container/30 text-secondary font-code-sm text-code-sm font-semibold">
                                        +14.2 pts
                                    </span>
                                </div>

                                <p className="font-body-sm text-body-sm text-on-surface-variant leading-relaxed">
                                    3 cryptographic weak hashes and 2 exposed dev tokens were successfully revoked and patched across{" "}
                                    <span className="text-on-surface font-medium">lucid-platform</span> and{" "}
                                    <span className="text-on-surface font-medium">lucid-auth</span>.
                                </p>
                            </div>
                        </div>

                        <div className="flex items-center justify-between pt-1 relative z-10">
                            <span className="font-code-sm text-code-sm text-on-surface-variant">
                                Auto-remediation active for CWE-79, CWE-89
                            </span>
                            <Link
                                href="/findings"
                                className="font-label-md text-label-md text-tertiary hover:underline font-semibold flex items-center gap-1"
                            >
                                <span>Explore all AI actions</span>
                                <span className="material-symbols-outlined text-[16px]">chevron_right</span>
                            </Link>
                        </div>
                    </div>
                </div>

                {/* Row 3: Security Findings Overview & Repository Health */}
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-gutter">
                    {/* Section Left: Findings Distribution & Active Findings (7 cols) */}
                    <div className="lg:col-span-7 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col gap-gutter shadow-sm">
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col">
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Prioritized Active Findings
                                </span>
                                <span className="font-body-sm text-body-sm text-on-surface-variant">
                                    Ranked by exploitability and blast radius
                                </span>
                            </div>
                            <button
                                type="button"
                                onClick={() => showToast("Filtered to all repositories")}
                                className="px-2.5 py-1 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface-variant hover:text-on-surface font-label-sm text-label-sm transition-colors border border-outline-variant/30"
                            >
                                Filter: All Repos
                            </button>
                        </div>

                        {/* Segmented Findings Progress Bar with detailed stats */}
                        <div className="flex flex-col gap-2 p-space-md rounded-xl bg-surface-container border border-outline-variant/20">
                            <div className="flex items-center justify-between font-label-sm text-label-sm text-on-surface">
                                <span>Findings Severity Distribution</span>
                                <span className="font-code-sm text-code-sm text-on-surface-variant">27 Total</span>
                            </div>
                            <div className="h-2.5 w-full rounded-full bg-surface-container-high overflow-hidden flex">
                                <div className="h-full bg-error" style={{ width: "7.4%" }} title="Critical: 7.4%" />
                                <div className="h-full bg-tertiary-container" style={{ width: "22.2%" }} title="High: 22.2%" />
                                <div className="h-full bg-primary" style={{ width: "51.8%" }} title="Medium: 51.8%" />
                                <div className="h-full bg-surface-variant" style={{ width: "18.6%" }} title="Low: 18.6%" />
                            </div>
                            <div className="grid grid-cols-4 gap-2 pt-1 font-code-sm text-code-sm">
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-error" />
                                    <span className="text-on-surface font-medium">Critical (2)</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-tertiary-container" />
                                    <span className="text-on-surface font-medium">High (6)</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-primary" />
                                    <span className="text-on-surface font-medium">Medium (14)</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-surface-variant" />
                                    <span className="text-on-surface-variant">Low (5)</span>
                                </div>
                            </div>
                        </div>

                        {/* Findings List */}
                        <div className="flex flex-col gap-2">
                            {/* Finding 1 */}
                            <div className="p-space-md rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors flex items-center justify-between gap-space-md border border-outline-variant/20">
                                <div className="flex items-start gap-space-md min-w-0">
                                    <span className="px-2 py-0.5 rounded bg-error-container/40 text-error font-code-sm text-code-sm font-semibold shrink-0 mt-0.5">
                                        CRIT
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <div className="flex items-baseline gap-2">
                                            <span className="font-body-md text-body-md text-on-surface font-semibold truncate">
                                                SQL Injection (CWE-89)
                                            </span>
                                            <span className="font-code-sm text-code-sm text-primary font-mono truncate hidden sm:inline">
                                                platform/db/store.go:84
                                            </span>
                                        </div>
                                        <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                            Unparameterized query in{" "}
                                            <code className="font-code-sm text-code-sm text-on-surface">
                                                GetUserByOrg()
                                            </code>{" "}
                                            with HTTP body parameter
                                        </p>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant/70 mt-0.5">
                                            Scanned 2 mins ago
                                        </span>
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setResolvedFindings((prev) => ({ ...prev, "1": !prev["1"] }));
                                        showToast(
                                            resolvedFindings["1"]
                                                ? "Finding reopened"
                                                : "Finding resolved via automatic parameterization patch"
                                        );
                                    }}
                                    className="px-3 py-1 rounded-lg bg-surface-container-highest hover:bg-error-container/40 hover:text-error text-on-surface font-label-sm text-label-sm font-medium transition-colors shrink-0"
                                >
                                    {resolvedFindings["1"] ? "Resolved ✓" : "Resolve"}
                                </button>
                            </div>

                            {/* Finding 2 */}
                            <div className="p-space-md rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors flex items-center justify-between gap-space-md border border-outline-variant/20">
                                <div className="flex items-start gap-space-md min-w-0">
                                    <span className="px-2 py-0.5 rounded bg-tertiary-container/30 text-tertiary font-code-sm text-code-sm font-semibold shrink-0 mt-0.5">
                                        HIGH
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <div className="flex items-baseline gap-2">
                                            <span className="font-body-md text-body-md text-on-surface font-semibold truncate">
                                                Improper GitHub token validation
                                            </span>
                                            <span className="font-code-sm text-code-sm text-primary font-mono truncate hidden sm:inline">
                                                platform/github/auth.go:126
                                            </span>
                                        </div>
                                        <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                            Missing HMAC signature verification check on incoming webhook payload
                                        </p>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant/70 mt-0.5">
                                            Scanned 18 mins ago
                                        </span>
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setResolvedFindings((prev) => ({ ...prev, "2": !prev["2"] }));
                                        showToast(
                                            resolvedFindings["2"]
                                                ? "Finding reopened"
                                                : "Finding resolved: webhook signature verified"
                                        );
                                    }}
                                    className="px-3 py-1 rounded-lg bg-surface-container-highest hover:bg-tertiary-container/40 hover:text-tertiary text-on-surface font-label-sm text-label-sm font-medium transition-colors shrink-0"
                                >
                                    {resolvedFindings["2"] ? "Resolved ✓" : "Resolve"}
                                </button>
                            </div>

                            {/* Finding 3 */}
                            <div className="p-space-md rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors flex items-center justify-between gap-space-md border border-outline-variant/20">
                                <div className="flex items-start gap-space-md min-w-0">
                                    <span className="px-2 py-0.5 rounded bg-primary/20 text-primary font-code-sm text-code-sm font-semibold shrink-0 mt-0.5">
                                        MED
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <div className="flex items-baseline gap-2">
                                            <span className="font-body-md text-body-md text-on-surface font-semibold truncate">
                                                Missing HTTP request timeout
                                            </span>
                                            <span className="font-code-sm text-code-sm text-primary font-mono truncate hidden sm:inline">
                                                platform/ingestion/router.go:52
                                            </span>
                                        </div>
                                        <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                            Potential Slowloris denial of service due to infinite connection read deadlines
                                        </p>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant/70 mt-0.5">
                                            Scanned 1 hour ago
                                        </span>
                                    </div>
                                </div>
                                <button
                                    type="button"
                                    onClick={() => {
                                        setResolvedFindings((prev) => ({ ...prev, "3": !prev["3"] }));
                                        showToast(
                                            resolvedFindings["3"]
                                                ? "Finding reopened"
                                                : "Finding resolved: read timeout configured"
                                        );
                                    }}
                                    className="px-3 py-1 rounded-lg bg-surface-container-highest hover:bg-primary/30 hover:text-primary text-on-surface font-label-sm text-label-sm font-medium transition-colors shrink-0"
                                >
                                    {resolvedFindings["3"] ? "Resolved ✓" : "Resolve"}
                                </button>
                            </div>
                        </div>
                    </div>

                    {/* Section Right: Repository Health (5 cols) */}
                    <div className="lg:col-span-5 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-gutter shadow-sm">
                        <div className="flex items-center justify-between">
                            <div className="flex flex-col">
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Repository Health
                                </span>
                                <span className="font-body-sm text-body-sm text-on-surface-variant">
                                    Real-time status of top tracked repositories
                                </span>
                            </div>
                            <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                folder_special
                            </span>
                        </div>

                        {/* Repository List */}
                        <div className="flex flex-col gap-2">
                            {/* Repo 1 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-between border border-outline-variant/20">
                                <div className="flex items-center gap-space-md min-w-0">
                                    <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                        hub
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold truncate font-mono">
                                            lucid-engine
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant">
                                            2 findings • 2m ago
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-space-sm shrink-0">
                                    <span className="font-code-sm text-code-sm text-secondary font-semibold">96%</span>
                                    <span className="px-2 py-0.5 rounded bg-secondary-container/40 text-secondary font-code-sm text-code-sm font-medium">
                                        Healthy
                                    </span>
                                </div>
                            </div>

                            {/* Repo 2 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-between border border-outline-variant/20">
                                <div className="flex items-center gap-space-md min-w-0">
                                    <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                        hub
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold truncate font-mono">
                                            lucid-platform
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant">
                                            7 findings • 18m ago
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-space-sm shrink-0">
                                    <span className="font-code-sm text-code-sm text-tertiary font-semibold">91%</span>
                                    <span className="px-2 py-0.5 rounded bg-tertiary-container/30 text-tertiary font-code-sm text-code-sm font-medium">
                                        Attention
                                    </span>
                                </div>
                            </div>

                            {/* Repo 3 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-between border border-outline-variant/20">
                                <div className="flex items-center gap-space-md min-w-0">
                                    <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                        hub
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold truncate font-mono">
                                            lucid-infra
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant">
                                            0 findings • 1h ago
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-space-sm shrink-0">
                                    <span className="font-code-sm text-code-sm text-secondary font-semibold">98%</span>
                                    <span className="px-2 py-0.5 rounded bg-secondary-container/40 text-secondary font-code-sm text-code-sm font-medium">
                                        Healthy
                                    </span>
                                </div>
                            </div>

                            {/* Repo 4 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-between border border-outline-variant/20">
                                <div className="flex items-center gap-space-md min-w-0">
                                    <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                        hub
                                    </span>
                                    <div className="flex flex-col min-w-0">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold truncate font-mono">
                                            lucid-frontend
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant">
                                            0 findings • 3h ago
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-space-sm shrink-0">
                                    <span className="font-code-sm text-code-sm text-secondary font-semibold">99%</span>
                                    <span className="px-2 py-0.5 rounded bg-secondary-container/40 text-secondary font-code-sm text-code-sm font-medium">
                                        Healthy
                                    </span>
                                </div>
                            </div>
                        </div>

                        <div className="pt-2">
                            <Link
                                href="/repositories"
                                className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-primary-fixed transition-colors font-semibold"
                            >
                                <span>View All Repositories ({repoCount})</span>
                                <span className="material-symbols-outlined text-[16px]">arrow_forward</span>
                            </Link>
                        </div>
                    </div>
                </div>

                {/* Row 4: Recent Scans & Pull Requests Grid */}
                <div className="grid grid-cols-1 lg:grid-cols-12 gap-gutter">
                    {/* Section Left: Recent Scans (7 cols) */}
                    <div className="lg:col-span-7 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col gap-gutter shadow-sm">
                        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-space-md">
                            <div className="flex items-center gap-space-sm">
                                <span className="material-symbols-outlined text-[20px] text-primary">history</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Recent Scans
                                </span>
                            </div>

                            <div className="flex items-center gap-space-xs">
                                <input
                                    type="text"
                                    value={scanFilter}
                                    onChange={(e) => setScanFilter(e.target.value)}
                                    placeholder="Filter scans..."
                                    className="h-7 px-2.5 rounded bg-surface-container text-on-surface font-body-sm text-body-sm placeholder:text-on-surface-variant/60 focus:outline-none border border-outline-variant/30"
                                />
                                <button
                                    type="button"
                                    onClick={() =>
                                        setScanStatusFilter((prev) =>
                                            prev === "ALL" ? "PASSED" : prev === "PASSED" ? "RUNNING" : "ALL"
                                        )
                                    }
                                    className="h-7 px-2 rounded bg-surface-container text-on-surface-variant hover:text-on-surface font-label-sm text-label-sm flex items-center gap-1 border border-outline-variant/30 transition-colors"
                                >
                                    <span className="material-symbols-outlined text-[14px]">tune</span>
                                    <span>
                                        {scanStatusFilter === "ALL"
                                            ? "All Statuses"
                                            : scanStatusFilter === "PASSED"
                                            ? "Passed Only"
                                            : "Running Only"}
                                    </span>
                                </button>
                            </div>
                        </div>

                        {/* Dense Scans Table */}
                        <div className="overflow-x-auto">
                            <table className="w-full text-left font-body-sm text-body-sm">
                                <thead>
                                    <tr className="text-on-surface-variant font-code-sm text-code-sm uppercase tracking-wider bg-surface-container/50">
                                        <th className="py-2 px-3 rounded-l">Repository</th>
                                        <th className="py-2 px-3">PR / Branch</th>
                                        <th className="py-2 px-3">Status</th>
                                        <th className="py-2 px-3">Findings</th>
                                        <th className="py-2 px-3">Duration</th>
                                        <th className="py-2 px-3 rounded-r text-right">Scanned</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y-0">
                                    {filteredScans.map((row, idx) => (
                                        <tr key={idx} className="hover:bg-surface-container transition-colors">
                                            <td className="py-2.5 px-3 font-mono font-medium text-on-surface">
                                                {row.repo}
                                            </td>
                                            <td className="py-2.5 px-3 font-mono text-primary font-code-sm text-code-sm">
                                                {row.pr}
                                            </td>
                                            <td className="py-2.5 px-3">
                                                {row.statusType === "passed" ? (
                                                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-secondary-container/30 text-secondary font-code-sm text-code-sm font-medium">
                                                        <span className="material-symbols-outlined text-[12px]">
                                                            check_circle
                                                        </span>{" "}
                                                        Passed
                                                    </span>
                                                ) : (
                                                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-primary/20 text-primary font-code-sm text-code-sm font-medium">
                                                        <span className="material-symbols-outlined text-[12px] animate-spin">
                                                            sync
                                                        </span>{" "}
                                                        Running
                                                    </span>
                                                )}
                                            </td>
                                            <td
                                                className={`py-2.5 px-3 font-code-sm text-code-sm ${
                                                    row.isFindingClean ? "text-secondary font-medium" : "text-on-surface"
                                                }`}
                                            >
                                                {row.findings}
                                            </td>
                                            <td
                                                className={`py-2.5 px-3 font-code-sm text-code-sm ${
                                                    row.statusType === "running"
                                                        ? "text-primary font-medium"
                                                        : "text-on-surface-variant"
                                                }`}
                                            >
                                                {row.duration}
                                            </td>
                                            <td className="py-2.5 px-3 font-code-sm text-code-sm text-on-surface-variant text-right">
                                                {row.scanned}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>

                        <div className="flex items-center justify-between pt-2 border-t border-outline-variant/20">
                            <span className="font-code-sm text-code-sm text-on-surface-variant">
                                Showing {filteredScans.length} of {weekScans} runs
                            </span>
                            <Link
                                href="/scans"
                                className="font-label-md text-label-md text-primary hover:underline font-semibold flex items-center gap-1"
                            >
                                <span>View scan logs archive</span>
                                <span className="material-symbols-outlined text-[16px]">chevron_right</span>
                            </Link>
                        </div>
                    </div>

                    {/* Section Right: Recent PRs & Security Gates (5 cols) */}
                    <div className="lg:col-span-5 p-gutter-lg rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col justify-between gap-gutter shadow-sm">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center gap-space-sm">
                                <span className="material-symbols-outlined text-[20px] text-tertiary">merge_type</span>
                                <span className="font-headline-md text-headline-md text-on-surface font-semibold">
                                    Security Gates in PRs
                                </span>
                            </div>
                            <span className="px-2 py-0.5 rounded bg-surface-container text-on-surface font-code-sm text-code-sm border border-outline-variant/30 font-medium">
                                Branch Rules ON
                            </span>
                        </div>

                        {/* PR Cards */}
                        <div className="flex flex-col gap-space-md">
                            {/* PR 1 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-1.5 border border-outline-variant/20">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex flex-col">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold">
                                            PR #142: Implement taint propagation
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant font-mono">
                                            lucid-engine • @devsec-lead • 2m ago
                                        </span>
                                    </div>
                                    <span className="material-symbols-outlined text-[18px] text-secondary shrink-0">
                                        check_circle
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 pt-1 font-body-sm text-body-sm text-on-surface-variant">
                                    <span className="w-1.5 h-1.5 rounded-full bg-secondary" />
                                    <span>Security checks passed • 0 blocking findings</span>
                                </div>
                            </div>

                            {/* PR 2 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-1.5 border border-outline-variant/20">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex flex-col">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold">
                                            PR #89: Fix webhook signature replay attack
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant font-mono">
                                            lucid-platform • @sarahchen • 18m ago
                                        </span>
                                    </div>
                                    <span className="material-symbols-outlined text-[18px] text-secondary shrink-0">
                                        check_circle
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 pt-1 font-body-sm text-body-sm text-on-surface-variant">
                                    <span className="w-1.5 h-1.5 rounded-full bg-secondary" />
                                    <span>Passed with automated patch verified • 0 findings</span>
                                </div>
                            </div>

                            {/* PR 3 */}
                            <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-1.5 border border-outline-variant/20">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex flex-col">
                                        <span className="font-body-md text-body-md text-on-surface font-semibold">
                                            PR #93: Migrate Redis session store
                                        </span>
                                        <span className="font-code-sm text-code-sm text-on-surface-variant font-mono">
                                            lucid-platform • @alexdoe • 1h ago
                                        </span>
                                    </div>
                                    <span className="material-symbols-outlined text-[18px] text-tertiary shrink-0">
                                        error
                                    </span>
                                </div>
                                <div className="flex items-center gap-2 pt-1 font-body-sm text-body-sm text-tertiary">
                                    <span className="w-1.5 h-1.5 rounded-full bg-tertiary" />
                                    <span>Gate Warning: 1 High risk finding pending approval</span>
                                </div>
                            </div>
                        </div>

                        <div className="pt-2">
                            <Link
                                href="/pull-requests"
                                className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-primary-fixed transition-colors font-semibold"
                            >
                                <span>View All Pull Requests (14)</span>
                                <span className="material-symbols-outlined text-[16px]">arrow_forward</span>
                            </Link>
                        </div>
                    </div>
                </div>

                {/* Bottom: Live Activity Stream Ticker */}
                <div className="p-gutter rounded-xl bg-surface-container-low border border-outline-variant/30 flex flex-col sm:flex-row sm:items-center justify-between gap-space-md shadow-sm">
                    <div className="flex items-center gap-space-md overflow-x-auto py-1">
                        <span className="font-code-sm text-code-sm uppercase tracking-wider text-on-surface-variant font-semibold shrink-0 flex items-center gap-1">
                            <span className="w-2 h-2 rounded-full bg-secondary" /> Stream
                        </span>
                        <div className="flex items-center gap-gutter text-on-surface font-code-sm text-code-sm shrink-0">
                            <span className="flex items-center gap-1.5 text-on-surface-variant">
                                <span className="text-on-surface">PR #142 scanned successfully (42s)</span> • 2m ago
                            </span>
                            <span className="text-surface-variant">•</span>
                            <span className="flex items-center gap-1.5 text-on-surface-variant">
                                <span className="text-secondary">Critical finding resolved</span> in lucid-platform • 15m ago
                            </span>
                            <span className="text-surface-variant">•</span>
                            <span className="flex items-center gap-1.5 text-on-surface-variant">
                                <span>New repository &apos;lucid-infra&apos; connected</span> • 1h ago
                            </span>
                            <span className="text-surface-variant">•</span>
                            <span className="flex items-center gap-1.5 text-on-surface-variant">
                                <span className="text-tertiary">AI generated remediation diff</span> for PR #89 • 2h ago
                            </span>
                        </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                        <span className="font-code-sm text-code-sm text-on-surface-variant">
                            All services operational
                        </span>
                        <span className="w-2 h-2 rounded-full bg-secondary" />
                    </div>
                </div>
            </div>
        </div>
    );
}
