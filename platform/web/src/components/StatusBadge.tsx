import { ScanStatus, VulnSeverity } from "@/lib/types";

export function StatusBadge({ status }: { status: ScanStatus | string }) {
    const normalized = (status || "").toUpperCase();

    switch (normalized) {
        case "COMPLETED":
        case "PASSED":
            return (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-emerald-500/20 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-400 font-mono">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    PASSED
                </span>
            );
        case "FAILED":
        case "BLOCKED":
            return (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-rose-500/20 bg-rose-500/10 px-2 py-0.5 text-[11px] font-medium text-rose-400 font-mono">
                    <span className="h-1.5 w-1.5 rounded-full bg-rose-400" />
                    BLOCKED
                </span>
            );
        case "SANDBOX_VERIFIED":
            return (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-cyan-500/20 bg-cyan-500/10 px-2 py-0.5 text-[11px] font-medium text-cyan-400 font-mono">
                    <span className="h-1.5 w-1.5 rounded-full bg-cyan-400" />
                    SANDBOX VERIFIED
                </span>
            );
        case "SCANNING":
        case "RUNNING":
        case "ANALYZING_AI":
        case "SANDBOXING":
            return (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-amber-500/20 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-400 font-mono">
                    <span className="h-1.5 w-1.5 rounded-full bg-amber-400 animate-pulse" />
                    {normalized}
                </span>
            );
        case "QUEUED":
        default:
            return (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-zinc-700 bg-zinc-800 px-2 py-0.5 text-[11px] font-medium text-zinc-300 font-mono">
                    <span className="h-1.5 w-1.5 rounded-full bg-zinc-400" />
                    {normalized || "PENDING"}
                </span>
            );
    }
}

export function SeverityBadge({ severity }: { severity: VulnSeverity | string }) {
    const normalized = (severity || "INFO").toUpperCase();

    const styles: Record<string, { bg: string; text: string; border: string }> = {
        CRITICAL: {
            bg: "bg-rose-500/10",
            text: "text-rose-400",
            border: "border-rose-500/30 ring-1 ring-rose-500/20",
        },
        HIGH: {
            bg: "bg-amber-500/10",
            text: "text-amber-400",
            border: "border-amber-500/30",
        },
        MEDIUM: {
            bg: "bg-indigo-500/10",
            text: "text-indigo-400",
            border: "border-indigo-500/30",
        },
        LOW: {
            bg: "bg-blue-500/10",
            text: "text-blue-400",
            border: "border-blue-500/20",
        },
        INFO: {
            bg: "bg-zinc-800",
            text: "text-zinc-400",
            border: "border-zinc-700",
        },
    };

    const style = styles[normalized] || styles.INFO;

    return (
        <span
            className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-mono font-semibold uppercase tracking-wider ${style.bg} ${style.text} ${style.border}`}
        >
            {normalized}
        </span>
    );
}
