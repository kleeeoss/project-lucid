// src/components/ScanRepoModal.tsx
"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import {
    Terminal,
    Search,
    Shield,
    GitBranch,
    AlertCircle,
    CheckCircle2,
    Loader2,
    X,
    ExternalLink
} from "lucide-react";

interface ScanRepoModalProps {
    isOpen: boolean;
    onClose: () => void;
    initialUrl?: string;
}

export function ScanRepoModal({ isOpen, onClose, initialUrl = "" }: ScanRepoModalProps) {
    const router = useRouter();
    const [repoUrl, setRepoUrl] = useState(initialUrl);
    const [branch, setBranch] = useState("main");
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [progressStep, setProgressStep] = useState<string>("");

    React.useEffect(() => {
        if (initialUrl) {
            setRepoUrl(initialUrl);
        }
    }, [initialUrl]);

    if (!isOpen) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!repoUrl.trim()) {
            setError("Please enter a valid GitHub repository URL.");
            return;
        }

        setIsLoading(true);
        setProgressStep("Cloning repository into ephemeral sandbox...");

        try {
            // Simulated progress steps during network clone and analysis
            const stepTimer1 = setTimeout(() => {
                setProgressStep("Parsing syntax trees & executing AST taint analysis...");
            }, 2500);

            const stepTimer2 = setTimeout(() => {
                setProgressStep("Generating React Flow AST graph & synthesizing remediation...");
            }, 5500);

            const res = await fetch("/api/scan", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    repo_url: repoUrl.trim(),
                    branch: branch.trim() || "main",
                }),
            });

            clearTimeout(stepTimer1);
            clearTimeout(stepTimer2);

            const data = await res.json();

            if (!res.ok || !data.success) {
                throw new Error(data.error || "Failed to scan repository.");
            }

            setProgressStep("Analysis complete! Redirecting to scan workbench...");
            setTimeout(() => {
                onClose();
                router.push(data.redirect_url);
            }, 600);
        } catch (err: any) {
            setError(err?.message || "An unexpected error occurred during scan execution.");
            setIsLoading(false);
        }
    };

    return (
        <div
            className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in"
            onClick={() => !isLoading && onClose()}
        >
            <div
                className="bg-zinc-950 border border-zinc-800 rounded-lg max-w-lg w-full p-5 shadow-2xl relative"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between pb-3 border-b border-zinc-800">
                    <div className="flex items-center gap-2">
                        <Terminal className="h-5 w-5 text-rose-400" />
                        <div>
                            <h2 className="text-sm font-semibold text-zinc-100 font-mono">
                                On-Demand Repository Scanner
                            </h2>
                            <p className="text-[11px] text-zinc-500">
                                Clone and analyze any public GitHub repo via Tree-sitter AST taint engine.
                            </p>
                        </div>
                    </div>
                    {!isLoading && (
                        <button
                            onClick={onClose}
                            className="p-1 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition"
                        >
                            <X className="h-4 w-4" />
                        </button>
                    )}
                </div>

                {/* Form */}
                <form onSubmit={handleSubmit} className="mt-4 space-y-4">
                    {error && (
                        <div className="rounded-md border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300 flex items-start gap-2">
                            <AlertCircle className="h-4 w-4 text-rose-400 shrink-0 mt-0.5" />
                            <span>{error}</span>
                        </div>
                    )}

                    <div className="space-y-1.5">
                        <label className="block text-xs font-mono font-medium text-zinc-300">
                            GitHub Repository HTTPS URL
                        </label>
                        <div className="relative">
                            <input
                                type="url"
                                required
                                disabled={isLoading}
                                value={repoUrl}
                                onChange={(e) => setRepoUrl(e.target.value)}
                                placeholder="https://github.com/owner/repository"
                                className="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs font-mono text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-700 focus:outline-none disabled:opacity-50"
                            />
                        </div>
                        <p className="text-[10px] text-zinc-500 font-mono">
                            Example: <span className="text-zinc-400">https://github.com/expressjs/express</span>
                        </p>
                    </div>

                    <div className="space-y-1.5">
                        <label className="block text-xs font-mono font-medium text-zinc-300">
                            Branch / Tag (Optional)
                        </label>
                        <div className="relative">
                            <input
                                type="text"
                                disabled={isLoading}
                                value={branch}
                                onChange={(e) => setBranch(e.target.value)}
                                placeholder="main"
                                className="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs font-mono text-zinc-100 placeholder:text-zinc-600 focus:border-zinc-700 focus:outline-none disabled:opacity-50"
                            />
                        </div>
                    </div>

                    {isLoading && (
                        <div className="rounded-md border border-zinc-800 bg-zinc-900/60 p-3.5 space-y-2">
                            <div className="flex items-center gap-2.5 text-xs font-mono text-emerald-400">
                                <Loader2 className="h-4 w-4 animate-spin text-emerald-400 shrink-0" />
                                <span className="font-semibold">{progressStep}</span>
                            </div>
                            <div className="w-full bg-zinc-800 h-1 rounded-full overflow-hidden">
                                <div className="bg-emerald-500 h-full w-2/3 animate-pulse" />
                            </div>
                        </div>
                    )}

                    {/* Actions */}
                    <div className="flex items-center justify-end gap-2 pt-2 border-t border-zinc-800">
                        <button
                            type="button"
                            disabled={isLoading}
                            onClick={onClose}
                            className="px-3 py-1.5 rounded-md bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium border border-zinc-800 transition disabled:opacity-50"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={isLoading}
                            className="px-4 py-1.5 rounded-md bg-rose-600 hover:bg-rose-500 text-white text-xs font-medium transition shadow-sm flex items-center gap-1.5 disabled:opacity-50"
                        >
                            {isLoading ? (
                                <>
                                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                    <span>Scanning...</span>
                                </>
                            ) : (
                                <>
                                    <Shield className="h-3.5 w-3.5" />
                                    <span>Trigger AST Scan</span>
                                </>
                            )}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
