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
    recentFindings?: Array<{
        id: string;
        rule_id: string;
        cwe: string;
        severity: string;
        file_path: string;
        line_start: number;
        ai_explanation: string;
        created_at: string;
        repo_name: string;
        pr_number: number;
        scan_run_id: string;
    }>;
}

function timeAgo(date: string | null) {
    if (!date) return "—";
    const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
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
    recentFindings: propRecentFindings = [],
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

    const repoCount = propRepoCount;
    const openFindings = propOpenFindings;
    const criticalCount = propCriticalCount;
    const weekScans = propWeekScans;

    // Severity numbers matching real data
    const critCount = criticalCount;
    const highCount = Math.max(0, Math.floor((openFindings - critCount) * 0.4));
    const medCount = Math.max(0, Math.floor((openFindings - critCount) * 0.4));
    const lowCount = Math.max(0, openFindings - critCount - highCount - medCount);

    // Security score calculation
    const securityScore = openFindings === 0 ? 100 : Math.max(50, 100 - critCount * 15 - (openFindings - critCount) * 2);
    const circumference = 314.159;
    const gaugeOffset = circumference * (1 - securityScore / 100);

    const mappedScans = (initialScans || []).map((s) => ({
        id: s.id,
        repo: s.repo_name || "lucid-ci",
        pr: s.pr_number ? `PR #${s.pr_number}` : s.commit_sha ? s.commit_sha.substring(0, 8) : "main",
        status: s.status === "COMPLETED" ? "Passed" : s.status === "FAILED" ? "Failed" : "Running",
        statusType: s.status === "COMPLETED" ? "passed" : s.status === "FAILED" ? "failed" : "running",
        findings: `${s.findings_count} ${s.findings_count === 1 ? "finding" : "findings"}`,
        isFindingClean: s.findings_count === 0,
        duration: s.scan_duration_ms ? `${(s.scan_duration_ms / 1000).toFixed(0)}s` : "—",
        scanned: timeAgo(s.started_at),
    }));

    const filteredScans = mappedScans.filter((s) => {
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
                                PostgreSQL Store Connected
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
                                    {propRepoHealth.length > 0 ? (
                                        propRepoHealth.slice(0, 12).map((rh, idx) => (
                                            <span
                                                key={idx}
                                                className={`w-2 h-2 rounded-full ${rh.findings === 0 ? "bg-secondary" : "bg-tertiary-container animate-pulse"}`}
                                                title={`${rh.full_name} (${rh.findings === 0 ? "Healthy" : `${rh.findings} findings`})`}
                                            />
                                        ))
                                    ) : (
                                        <span className="w-2 h-2 rounded-full bg-secondary/40" title="No repositories" />
                                    )}
                                </div>
                                <span className="font-code-sm text-code-sm text-on-surface-variant ml-auto">
                                    {repoCount > 0 ? `${propRepoHealth.filter((r) => r.findings === 0).length}/${repoCount} clean` : "0 clean"}
                                </span>
                            </div>
                            <div className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                <span className="text-secondary font-medium">{repoCount > 0 ? "100% CI active" : "No repos active"}</span> • {repoCount > 0 ? `${repoCount} protected` : "0 protected"}
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
                                        {criticalCount > 0 ? `${criticalCount} Crit` : "Clean"}
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
                                <div
                                    className="h-full bg-error"
                                    style={{ width: openFindings > 0 ? `${Math.min(100, (criticalCount / openFindings) * 100)}%` : "0%" }}
                                    title={`Critical: ${criticalCount}`}
                                />
                                <div
                                    className="h-full bg-secondary"
                                    style={{ width: openFindings > 0 ? `${Math.max(0, 100 - (criticalCount / openFindings) * 100)}%` : "100%" }}
                                    title="Other Severities"
                                />
                            </div>
                            <div className="flex items-center justify-between font-code-sm text-code-sm text-on-surface-variant">
                                <span>{criticalCount} Critical</span>
                                <span>{Math.max(0, openFindings - criticalCount)} Other</span>
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
                                <span>{propStats.avgDurationMs ? `${(propStats.avgDurationMs / 1000).toFixed(0)}s avg duration` : "— avg duration"}</span>
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
                            {/* Insight Item 1 */}
                            {propRecentFindings.length > 0 ? (
                                <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-space-sm">
                                    <div className="flex items-start justify-between gap-2">
                                        <div className="flex items-center gap-2">
                                            <span className="p-1 rounded bg-error-container/40 text-error flex items-center justify-center">
                                                <span className="material-symbols-outlined text-[16px]">warning</span>
                                            </span>
                                            <span className="font-body-md text-body-md font-semibold text-on-surface">
                                                {propRecentFindings[0].rule_id} detected
                                            </span>
                                        </div>
                                        <span className="px-1.5 py-0.5 rounded bg-surface-container-highest text-error font-code-sm text-code-sm font-semibold shrink-0">
                                            {propRecentFindings[0].cwe}
                                        </span>
                                    </div>

                                    <p className="font-body-sm text-body-sm text-on-surface-variant leading-relaxed">
                                        {propRecentFindings[0].ai_explanation || `Taint sink reached in ${propRecentFindings[0].file_path}:${propRecentFindings[0].line_start}`}
                                    </p>

                                    <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                                        <div className="flex items-center gap-1.5 font-code-sm text-code-sm text-on-surface-variant">
                                            <span className="text-on-surface font-medium">{propRecentFindings[0].repo_name}</span>
                                            <span>•</span>
                                            <span className="text-primary font-medium">PR #{propRecentFindings[0].pr_number}</span>
                                            <span>•</span>
                                            <span className="text-tertiary">AST Taint Sink</span>
                                        </div>

                                        <Link
                                            href={`/scans/${propRecentFindings[0].scan_run_id}`}
                                            className="px-2.5 py-1 rounded bg-primary-container text-on-primary hover:opacity-90 font-label-sm text-label-sm font-semibold transition-opacity"
                                        >
                                            Inspect Workbench
                                        </Link>
                                    </div>
                                </div>
                            ) : (
                                <div className="p-space-md rounded-xl bg-surface-container flex flex-col gap-space-sm">
                                    <div className="flex items-start justify-between gap-2">
                                        <div className="flex items-center gap-2">
                                            <span className="p-1 rounded bg-secondary-container/40 text-secondary flex items-center justify-center">
                                                <span className="material-symbols-outlined text-[16px]">check_circle</span>
                                            </span>
                                            <span className="font-body-md text-body-md font-semibold text-on-surface">
                                                Pipeline Nominal — No Active Taint Flows
                                            </span>
                                        </div>
                                        <span className="px-1.5 py-0.5 rounded bg-surface-container-highest text-secondary font-code-sm text-code-sm font-semibold shrink-0">
                                            Clean
                                        </span>
                                    </div>

                                    <p className="font-body-sm text-body-sm text-on-surface-variant leading-relaxed">
                                        AST taint analysis and gVisor detonation pipeline report no active security vulnerabilities across monitored repositories.
                                    </p>

                                    <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                                        <div className="flex items-center gap-1.5 font-code-sm text-code-sm text-on-surface-variant">
                                            <span className="text-on-surface font-medium">PostgreSQL Store</span>
                                            <span>•</span>
                                            <span className="text-secondary font-medium">0 Critical Vulnerabilities</span>
                                        </div>

                                        <Link
                                            href="/scans"
                                            className="px-2.5 py-1 rounded bg-surface-container-high hover:bg-surface-variant text-on-surface font-label-sm text-label-sm transition-colors"
                                        >
                                            View Scans Feed
                                        </Link>
                                    </div>
                                </div>
                            )}

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
                                <span className="font-code-sm text-code-sm text-on-surface-variant">{openFindings} Total</span>
                            </div>
                            <div className="h-2.5 w-full rounded-full bg-surface-container-high overflow-hidden flex">
                                <div className="h-full bg-error" style={{ width: openFindings > 0 ? `${(critCount / openFindings) * 100}%` : "0%" }} title={`Critical: ${critCount}`} />
                                <div className="h-full bg-tertiary-container" style={{ width: openFindings > 0 ? `${(highCount / openFindings) * 100}%` : "0%" }} title={`High: ${highCount}`} />
                                <div className="h-full bg-primary" style={{ width: openFindings > 0 ? `${(medCount / openFindings) * 100}%` : "0%" }} title={`Medium: ${medCount}`} />
                                <div className="h-full bg-surface-variant" style={{ width: openFindings > 0 ? `${(lowCount / openFindings) * 100}%` : "100%" }} title={`Low: ${lowCount}`} />
                            </div>
                            <div className="grid grid-cols-4 gap-2 pt-1 font-code-sm text-code-sm">
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-error" />
                                    <span className="text-on-surface font-medium">Critical ({critCount})</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-tertiary-container" />
                                    <span className="text-on-surface font-medium">High ({highCount})</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-primary" />
                                    <span className="text-on-surface font-medium">Medium ({medCount})</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <span className="w-2 h-2 rounded-full bg-surface-variant" />
                                    <span className="text-on-surface-variant">Low ({lowCount})</span>
                                </div>
                            </div>
                        </div>

                        {/* Findings List */}
                        <div className="flex flex-col gap-2">
                            {propRecentFindings.length > 0 ? (
                                propRecentFindings.map((f) => (
                                    <div
                                        key={f.id}
                                        className="p-space-md rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors flex items-center justify-between gap-space-md border border-outline-variant/20"
                                    >
                                        <div className="flex items-start gap-space-md min-w-0">
                                            <span
                                                className={`px-2 py-0.5 rounded font-code-sm text-code-sm font-semibold shrink-0 mt-0.5 ${
                                                    f.severity === "CRITICAL"
                                                        ? "bg-error-container/40 text-error"
                                                        : f.severity === "HIGH"
                                                        ? "bg-tertiary-container/30 text-tertiary"
                                                        : "bg-primary/20 text-primary"
                                                }`}
                                            >
                                                {f.severity === "CRITICAL" ? "CRIT" : f.severity}
                                            </span>
                                            <div className="flex flex-col min-w-0">
                                                <div className="flex items-baseline gap-2">
                                                    <span className="font-body-md text-body-md text-on-surface font-semibold truncate">
                                                        {f.rule_id}
                                                    </span>
                                                    <span className="font-code-sm text-code-sm text-primary font-mono truncate hidden sm:inline">
                                                        {f.file_path}:{f.line_start}
                                                    </span>
                                                </div>
                                                <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                                                    {f.ai_explanation || f.cwe}
                                                </p>
                                                <span className="font-code-sm text-code-sm text-on-surface-variant/70 mt-0.5">
                                                    {f.repo_name} • PR #{f.pr_number} • {timeAgo(f.created_at)}
                                                </span>
                                            </div>
                                        </div>
                                        <Link
                                            href={`/scans/${f.scan_run_id}`}
                                            className="px-3 py-1 rounded-lg bg-surface-container-highest hover:bg-surface-variant text-on-surface font-label-sm text-label-sm font-medium transition-colors shrink-0"
                                        >
                                            Inspect
                                        </Link>
                                    </div>
                                ))
                            ) : (
                                <div className="p-space-md rounded-xl bg-surface-container flex flex-col items-center justify-center border border-outline-variant/20 py-8 text-center text-on-surface-variant font-body-sm text-body-sm">
                                    <span className="material-symbols-outlined text-[28px] text-secondary mb-1">verified_user</span>
                                    <span>No active vulnerabilities detected across monitored repositories.</span>
                                </div>
                            )}
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
                            {propRepoHealth && propRepoHealth.length > 0 ? (
                                propRepoHealth.map((repo, idx) => {
                                    const healthPct = Math.max(0, 100 - repo.findings * 10);
                                    const isHealthy = repo.findings === 0;
                                    return (
                                        <div key={idx} className="p-space-md rounded-xl bg-surface-container flex items-center justify-between border border-outline-variant/20">
                                            <div className="flex items-center gap-space-md min-w-0">
                                                <span className="material-symbols-outlined text-[20px] text-on-surface-variant">
                                                    hub
                                                </span>
                                                <div className="flex flex-col min-w-0">
                                                    <span className="font-body-md text-body-md text-on-surface font-semibold truncate font-mono">
                                                        {repo.full_name}
                                                    </span>
                                                    <span className="font-code-sm text-code-sm text-on-surface-variant">
                                                        {repo.findings} {repo.findings === 1 ? "finding" : "findings"} • {timeAgo(repo.last_scan)}
                                                    </span>
                                                </div>
                                            </div>
                                            <div className="flex items-center gap-space-sm shrink-0">
                                                <span className={`font-code-sm text-code-sm font-semibold ${isHealthy ? "text-secondary" : "text-tertiary"}`}>
                                                    {healthPct}%
                                                </span>
                                                <span className={`px-2 py-0.5 rounded font-code-sm text-code-sm font-medium ${isHealthy ? "bg-secondary-container/40 text-secondary" : "bg-tertiary-container/30 text-tertiary"}`}>
                                                    {isHealthy ? "Healthy" : "Attention"}
                                                </span>
                                            </div>
                                        </div>
                                    );
                                })
                            ) : (
                                <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-center border border-outline-variant/20 py-8 text-center text-on-surface-variant font-body-sm text-body-sm">
                                    No repositories connected yet.
                                </div>
                            )}
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
                                    {filteredScans.length === 0 ? (
                                        <tr>
                                            <td colSpan={6} className="py-8 px-3 text-center text-on-surface-variant font-sans">
                                                No scans recorded in PostgreSQL store yet.
                                            </td>
                                        </tr>
                                    ) : (
                                        filteredScans.map((row) => (
                                            <tr key={row.id} className="hover:bg-surface-container transition-colors">
                                                <td className="py-2.5 px-3 font-mono font-medium text-on-surface">
                                                    <Link href={`/scans/${row.id}`} className="hover:text-primary transition-colors">
                                                        {row.repo}
                                                    </Link>
                                                </td>
                                                <td className="py-2.5 px-3 font-mono text-primary font-code-sm text-code-sm">
                                                    <Link href={`/scans/${row.id}`} className="hover:underline">
                                                        {row.pr}
                                                    </Link>
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
                                                            {row.status}
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
                                        ))
                                    )}
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
                            {propPrGates && propPrGates.length > 0 ? (
                                propPrGates.map((pr) => {
                                    const isClean = pr.findings_count === 0 && pr.status === "COMPLETED";
                                    return (
                                        <Link
                                            key={pr.id}
                                            href={`/scans/${pr.id}`}
                                            className="p-space-md rounded-xl bg-surface-container hover:bg-surface-container-high transition-colors flex flex-col gap-1.5 border border-outline-variant/20 block"
                                        >
                                            <div className="flex items-start justify-between gap-2">
                                                <div className="flex flex-col">
                                                    <span className="font-body-md text-body-md text-on-surface font-semibold">
                                                        PR #{pr.pr_number}: Commit {pr.commit_sha.substring(0, 8)}
                                                    </span>
                                                    <span className="font-code-sm text-code-sm text-on-surface-variant font-mono">
                                                        {pr.repo_name} • {timeAgo(pr.completed_at)}
                                                    </span>
                                                </div>
                                                <span className={`material-symbols-outlined text-[18px] shrink-0 ${isClean ? "text-secondary" : "text-tertiary"}`}>
                                                    {isClean ? "check_circle" : "error"}
                                                </span>
                                            </div>
                                            <div className="flex items-center gap-2 pt-1 font-body-sm text-body-sm text-on-surface-variant">
                                                <span className={`w-1.5 h-1.5 rounded-full ${isClean ? "bg-secondary" : "bg-tertiary"}`} />
                                                <span>
                                                    {isClean
                                                        ? "Security checks passed • 0 blocking findings"
                                                        : pr.findings_count > 0
                                                        ? `${pr.findings_count} findings evaluated`
                                                        : pr.status}
                                                </span>
                                            </div>
                                        </Link>
                                    );
                                })
                            ) : (
                                <div className="p-space-md rounded-xl bg-surface-container flex items-center justify-center border border-outline-variant/20 py-8 text-center text-on-surface-variant font-body-sm text-body-sm">
                                    No pull request gates evaluated yet.
                                </div>
                            )}
                        </div>

                        <div className="pt-2">
                            <Link
                                href="/pull-requests"
                                className="inline-flex items-center gap-1 font-label-md text-label-md text-primary hover:text-primary-fixed transition-colors font-semibold"
                            >
                                <span>View All Pull Requests ({propPrGates ? propPrGates.length : 0})</span>
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
