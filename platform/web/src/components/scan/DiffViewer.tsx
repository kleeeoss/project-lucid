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

    const handleCopy = () => {
        navigator.clipboard.writeText(patchCode);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    const originalLines = originalCode.trim().split("\n");
    const patchLines = patchCode.trim().split("\n");

    return (
        <div className="rounded-xl border border-zinc-800 bg-[#090b10] overflow-hidden font-mono text-xs shadow-xl">
            {/* File Header Bar */}
            <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-2.5">
                <div className="flex items-center gap-2 text-zinc-300">
                    <FileCode2 className="h-4 w-4 text-zinc-500" />
                    <span className="font-semibold">{filePath}</span>
                    <span className="text-zinc-600 text-[11px]">(around line {lineStart})</span>
                </div>

                <button
                    onClick={handleCopy}
                    className="inline-flex items-center gap-1.5 rounded-md border border-zinc-700 bg-zinc-800 px-2.5 py-1 text-[11px] text-zinc-200 hover:bg-zinc-700 hover:text-white transition"
                >
                    {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                    {copied ? "Copied Patch" : "Copy Patch"}
                </button>
            </div>

            {/* Code Diff Display */}
            <div className="divide-y divide-zinc-900 overflow-x-auto">
                {/* Red: Removed / Vulnerable Code */}
                {originalLines.map((line, i) => (
                    <div key={`orig-${i}`} className="flex items-center bg-rose-950/20 text-rose-200 hover:bg-rose-950/30">
                        <span className="w-12 select-none py-1 text-center text-[10px] text-rose-500/60 font-mono">
                            {lineStart + i}
                        </span>
                        <span className="w-6 select-none text-center font-bold text-rose-400">-</span>
                        <pre className="py-1 pr-4 text-xs">{line}</pre>
                    </div>
                ))}

                {/* Green: Added / Remediated Patch Code */}
                {patchLines.map((line, i) => (
                    <div key={`patch-${i}`} className="flex items-center bg-emerald-950/20 text-emerald-200 hover:bg-emerald-950/30">
                        <span className="w-12 select-none py-1 text-center text-[10px] text-emerald-500/60 font-mono">
                            {lineStart + i}
                        </span>
                        <span className="w-6 select-none text-center font-bold text-emerald-400">+</span>
                        <pre className="py-1 pr-4 text-xs font-semibold">{line}</pre>
                    </div>
                ))}
            </div>
        </div>
    );
}