// src/components/scan/DiffViewer.tsx
"use client";

import React, { useState } from "react";
import { Check, Copy, FileCode2 } from "lucide-react";

interface DiffViewerProps {
    originalCode: string;
    patchCode: string;
    filePath: string;
    lineStart: number;
}

export function DiffViewer({ originalCode, patchCode, filePath, lineStart }: DiffViewerProps) {
    const [copied, setCopied] = useState(false);

    const hasValidPatch = Boolean(
        patchCode &&
        patchCode.trim() &&
        !patchCode.trim().startsWith("// No automated patch")
    );

    const handleCopy = async () => {
        if (!hasValidPatch) return;
        try {
            if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(patchCode);
                setCopied(true);
                setTimeout(() => setCopied(false), 2000);
            }
        } catch (err) {
            console.warn("Failed to copy patch to clipboard:", err);
        }
    };

    const originalLines = originalCode ? originalCode.trim().split("\n") : [];
    const patchLines = hasValidPatch ? patchCode.trim().split("\n") : [];

    return (
        <div className="rounded-lg border border-zinc-800 bg-[#09090b] overflow-hidden font-mono text-xs shadow-xl">
            {/* File Header Bar */}
            <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/90 px-4 py-2.5">
                <div className="flex items-center gap-2 text-zinc-200">
                    <FileCode2 className="h-4 w-4 text-zinc-400" />
                    <span className="font-semibold">{filePath}</span>
                    <span className="text-zinc-500 text-[11px]">(line {lineStart})</span>
                </div>

                {hasValidPatch && (
                    <button
                        onClick={handleCopy}
                        className="inline-flex items-center gap-1.5 rounded-md border border-zinc-700 bg-zinc-800/90 px-2.5 py-1 text-[11px] font-sans text-zinc-200 hover:bg-zinc-700 hover:text-white transition"
                    >
                        {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5 text-zinc-400" />}
                        {copied ? "Copied Patch" : "Copy Patch"}
                    </button>
                )}
            </div>

            {/* Code Sections Display */}
            <div className="divide-y divide-zinc-900 overflow-x-auto">
                {/* Vulnerable Code Header & Lines */}
                <div className="flex items-center justify-between px-4 py-1.5 bg-rose-950/30 border-b border-rose-900/30 text-[10px] font-mono font-semibold uppercase tracking-wider text-rose-400">
                    <span>Original Vulnerable Code</span>
                    <span className="text-rose-500/70">{originalLines.length} lines</span>
                </div>
                {originalLines.map((line, i) => (
                    <div key={`orig-${i}`} className="flex items-center bg-rose-950/15 text-rose-200 hover:bg-rose-950/25 border-l-2 border-l-rose-500/80 transition-colors">
                        <span className="w-12 select-none py-1 text-center text-[10px] text-rose-400/60 font-mono bg-rose-950/20">
                            {lineStart + i}
                        </span>
                        <span className="w-6 select-none text-center font-bold text-rose-400">-</span>
                        <pre className="py-1 pr-4 text-xs whitespace-pre overflow-x-auto">{line}</pre>
                    </div>
                ))}

                {/* Remediation Patch Header & Lines */}
                <div className="flex items-center justify-between px-4 py-1.5 bg-emerald-950/30 border-b border-emerald-900/30 text-[10px] font-mono font-semibold uppercase tracking-wider text-emerald-400">
                    <span>Suggested Remediation Patch</span>
                    <span className="text-emerald-500/70">{patchLines.length} lines</span>
                </div>
                {hasValidPatch ? (
                    patchLines.map((line, i) => (
                        <div key={`patch-${i}`} className="flex items-center bg-emerald-950/15 text-emerald-200 hover:bg-emerald-950/25 border-l-2 border-l-emerald-500/80 transition-colors">
                            <span className="w-12 select-none py-1 text-center text-[10px] text-emerald-400/60 font-mono bg-emerald-950/20">
                                +{i + 1}
                            </span>
                            <span className="w-6 select-none text-center font-bold text-emerald-400">+</span>
                            <pre className="py-1 pr-4 text-xs font-semibold whitespace-pre overflow-x-auto">{line}</pre>
                        </div>
                    ))
                ) : (
                    <div className="p-8 text-center text-zinc-500 font-sans text-xs">
                        No automated remediation patch was generated for this vulnerability finding.
                    </div>
                )}
            </div>
        </div>
    );
}
