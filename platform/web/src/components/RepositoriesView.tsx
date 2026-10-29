// src/components/RepositoriesView.tsx
"use client";

import React, { useState, useMemo } from "react";
import Link from "next/link";
import {
    GitFork,
    Shield,
    Clock,
    ArrowRight,
    Search,
    Filter,
    Plus,
    ExternalLink,
    Terminal,
    Sparkles,
    AlertTriangle,
    CheckCircle2
} from "lucide-react";
import { ScanRepoModal } from "@/components/ScanRepoModal";

export interface RepositoryItem {
    id: number | string;
    full_name: string;
    default_branch: string;
    scan_count: number;
    total_findings: number;
    last_scan_at: string | null;
    created_at?: string;
}

export interface RepositoriesViewProps {
    repos: RepositoryItem[];
}

function timeAgo(date: string | null) {
    if (!date) return "Never scanned";
    const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
}

export function RepositoriesView({ repos }: RepositoriesViewProps) {
    const [search, setSearch] = useState("");
    const [healthFilter, setHealthFilter] = useState<"ALL" | "CLEAN" | "AT_RISK">("ALL");
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [selectedRepoUrl, setSelectedRepoUrl] = useState("");

    const filteredRepos = useMemo(() => {
        return repos.filter((repo) => {
            const matchesSearch =
                repo.full_name.toLowerCase().includes(search.toLowerCase()) ||
                repo.default_branch.toLowerCase().includes(search.toLowerCase());

            if (!matchesSearch) return false;

            if (healthFilter === "CLEAN") {
                return repo.total_findings === 0;
            }
            if (healthFilter === "AT_RISK") {
                return repo.total_findings > 0;
            }
            return true;
        });
    }, [repos, search, healthFilter]);

    const handleOpenModal = (repoUrl: string = "") => {
        setSelectedRepoUrl(repoUrl);
        setIsModalOpen(true);
    };

    return (
        <div className="space-y-6 max-w-7xl mx-auto w-full">
            {/* Quick Scan Public Repo Modal */}
            <ScanRepoModal
                isOpen={isModalOpen}
                onClose={() => {
                    setIsModalOpen(false);
                    setSelectedRepoUrl("");
                }}
                initialUrl={selectedRepoUrl}
            />

            {/* Header & Primary Action */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800/80 pb-5">
                <div className="space-y-1">
                    <div className="flex items-center gap-3">
                        <h1 className="text-xl font-semibold text-zinc-100 tracking-tight">
                            Tracked Repositories
                        </h1>
                        <span className="px-2 py-0.5 rounded-md bg-zinc-900 border border-zinc-800 text-zinc-400 font-mono text-[11px] font-medium">
                            {repos.length} Repositories Monitored
                        </span>
                    </div>
                    <p className="text-xs text-zinc-400">
                        Manage repositories monitored by Lucid-CI for AST taint analysis, gVisor sandboxing, and automated PR gates.
                    </p>
                </div>
                <button
                    type="button"
                    onClick={() => handleOpenModal("")}
                    className="flex items-center justify-center gap-2 rounded-md bg-rose-600 px-3.5 py-1.5 text-xs font-medium text-white hover:bg-rose-500 transition shadow-sm self-start sm:self-auto"
                >
                    <Plus className="h-4 w-4" />
                    <span>Add Repository</span>
                </button>
            </div>

            {/* Search and Filters */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
                <div className="flex items-center gap-2 rounded-md border border-zinc-800 bg-zinc-900/80 px-3 py-1.5 text-xs text-zinc-100 flex-1 max-w-md focus-within:border-zinc-700 transition">
                    <Search className="h-4 w-4 text-zinc-500 shrink-0" />
                    <input
                        type="text"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder="Search repository or branch..."
                        className="bg-transparent outline-none w-full text-xs font-mono text-zinc-100 placeholder:text-zinc-500"
                    />
                    {search && (
                        <button
                            type="button"
                            onClick={() => setSearch("")}
                            className="text-[10px] font-mono text-zinc-500 hover:text-zinc-300"
                        >
                            CLEAR
                        </button>
                    )}
                </div>

                <div className="flex items-center gap-2">
                    <div className="flex items-center p-0.5 rounded-md bg-zinc-900 border border-zinc-800 text-xs">
                        <button
                            type="button"
                            onClick={() => setHealthFilter("ALL")}
                            className={`px-2.5 py-1 rounded text-xs transition-colors ${
                                healthFilter === "ALL"
                                    ? "bg-zinc-800 text-white font-medium shadow-sm"
                                    : "text-zinc-400 hover:text-zinc-200"
                            }`}
                        >
                            All ({repos.length})
                        </button>
                        <button
                            type="button"
                            onClick={() => setHealthFilter("CLEAN")}
                            className={`px-2.5 py-1 rounded text-xs transition-colors ${
                                healthFilter === "CLEAN"
                                    ? "bg-zinc-800 text-emerald-400 font-medium shadow-sm"
                                    : "text-zinc-400 hover:text-zinc-200"
                            }`}
                        >
                            Clean
                        </button>
                        <button
                            type="button"
                            onClick={() => setHealthFilter("AT_RISK")}
                            className={`px-2.5 py-1 rounded text-xs transition-colors ${
                                healthFilter === "AT_RISK"
                                    ? "bg-zinc-800 text-rose-400 font-medium shadow-sm"
                                    : "text-zinc-400 hover:text-zinc-200"
                            }`}
                        >
                            At Risk
                        </button>
                    </div>
                </div>
            </div>

            {/* Repository List */}
            {filteredRepos.length === 0 ? (
                <div className="rounded-lg border border-zinc-800 bg-zinc-950/80 p-12 text-center">
                    <GitFork className="h-10 w-10 text-zinc-600 mx-auto mb-3" />
                    <h3 className="text-sm font-semibold text-zinc-200 font-mono">
                        {search ? "No matching repositories found" : "No repositories connected yet"}
                    </h3>
                    <p className="mt-1 text-xs text-zinc-400 max-w-sm mx-auto">
                        {search
                            ? `No repositories matched "${search}". Try adjusting your search query.`
                            : "Click below to trigger an on-demand scan of any public GitHub repository or configure webhook gates."}
                    </p>
                    <div className="mt-4">
                        <button
                            type="button"
                            onClick={() => handleOpenModal("")}
                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-rose-600 hover:bg-rose-500 text-white text-xs font-medium transition shadow-sm"
                        >
                            <Sparkles className="h-3.5 w-3.5" />
                            <span>Scan Public Repository</span>
                        </button>
                    </div>
                </div>
            ) : (
                <div className="grid grid-cols-1 gap-3">
                    {filteredRepos.map((repo) => {
                        const healthPct = Math.max(0, 100 - repo.total_findings * 10);
                        const isClean = repo.total_findings === 0;

                        return (
                            <div
                                key={repo.id}
                                className="rounded-lg border border-zinc-800 bg-zinc-950/80 p-4 hover:border-zinc-700/80 transition-all shadow-sm"
                            >
                                <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
                                    {/* Left metadata */}
                                    <div className="flex items-start gap-3.5">
                                        <div className="h-9 w-9 rounded-md bg-zinc-900 border border-zinc-800 flex items-center justify-center shrink-0">
                                            <GitFork className="h-4 w-4 text-rose-400" />
                                        </div>
                                        <div className="space-y-1">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <a
                                                    href={`https://github.com/${repo.full_name}`}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-sm font-semibold font-mono text-zinc-100 hover:text-rose-400 transition flex items-center gap-1.5"
                                                >
                                                    <span>{repo.full_name}</span>
                                                    <ExternalLink className="h-3 w-3 text-zinc-500" />
                                                </a>

                                                <span className="px-2 py-0.5 rounded bg-zinc-900 border border-zinc-800 text-[10px] font-mono text-zinc-400">
                                                    {repo.default_branch || "main"}
                                                </span>

                                                {isClean ? (
                                                    <span className="px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 font-mono text-[10px] flex items-center gap-1">
                                                        <CheckCircle2 className="h-2.5 w-2.5" />
                                                        PASS
                                                    </span>
                                                ) : (
                                                    <span className="px-2 py-0.5 rounded bg-rose-500/10 border border-rose-500/20 text-rose-400 font-mono text-[10px] flex items-center gap-1">
                                                        <AlertTriangle className="h-2.5 w-2.5" />
                                                        {repo.total_findings} ISSUES
                                                    </span>
                                                )}
                                            </div>

                                            <div className="flex flex-wrap items-center gap-3 text-xs text-zinc-400 font-mono">
                                                <span className="flex items-center gap-1">
                                                    <Terminal className="h-3 w-3 text-zinc-500" />
                                                    {repo.scan_count} {repo.scan_count === 1 ? "scan" : "scans"}
                                                </span>
                                                <span>•</span>
                                                <span className="flex items-center gap-1">
                                                    <Clock className="h-3 w-3 text-zinc-500" />
                                                    {timeAgo(repo.last_scan_at)}
                                                </span>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Right: Health Metrics and Actions */}
                                    <div className="flex items-center justify-between lg:justify-end gap-5 pt-3 lg:pt-0 border-t lg:border-t-0 border-zinc-800/80">
                                        <div className="flex flex-col items-start lg:items-end gap-1">
                                            <div className="flex items-center gap-2">
                                                <div className="w-24 h-1.5 rounded-full bg-zinc-800 overflow-hidden">
                                                    <div
                                                        className={`h-full rounded-full ${
                                                            isClean
                                                                ? "bg-emerald-500"
                                                                : healthPct >= 70
                                                                ? "bg-amber-500"
                                                                : "bg-rose-500"
                                                        }`}
                                                        style={{ width: `${healthPct}%` }}
                                                    />
                                                </div>
                                                <span
                                                    className={`text-xs font-mono font-medium ${
                                                        isClean
                                                            ? "text-emerald-400"
                                                            : healthPct >= 70
                                                            ? "text-amber-400"
                                                            : "text-rose-400"
                                                    }`}
                                                >
                                                    {healthPct}%
                                                </span>
                                            </div>
                                            <span className="text-[10px] font-mono text-zinc-500">
                                                {isClean
                                                    ? "Zero vulnerabilities"
                                                    : `${repo.total_findings} open vulnerabilities`}
                                            </span>
                                        </div>

                                        <div className="flex items-center gap-2">
                                            <button
                                                type="button"
                                                onClick={() => handleOpenModal(`https://github.com/${repo.full_name}`)}
                                                className="flex items-center gap-1.5 h-8 px-2.5 rounded-md bg-zinc-900 hover:bg-zinc-800 text-zinc-200 text-xs font-mono font-medium transition border border-zinc-800"
                                                title="Trigger AST scan on this repository"
                                            >
                                                <Sparkles className="h-3 w-3 text-rose-400" />
                                                <span>Quick Scan</span>
                                            </button>

                                            <Link
                                                href={`/scans`}
                                                className="flex items-center justify-center h-8 w-8 rounded-md bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 border border-zinc-800 transition"
                                                title="View Scan Runs"
                                            >
                                                <ArrowRight className="h-3.5 w-3.5" />
                                            </Link>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
