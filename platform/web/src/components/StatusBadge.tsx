import { ScanStatus, VulnSeverity } from "@/lib/types";

export function StatusBadge({ status }: { status: ScanStatus }) {
    const styles: Record<ScanStatus, string> = {
        QUEUED: "bg-amber-500/10 text-amber-400 border-amber-500/20",
        SCANNING: "bg-sky-500/10 text-sky-400 border-sky-500/30 animate-pulse",
        ANALYZING_AI: "bg-purple-500/10 text-purple-400 border-purple-500/20",
        SANDBOXING: "bg-indigo-500/10 text-indigo-400 border-indigo-500/20",
        COMPLETED: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
        FAILED: "bg-rose-500/10 text-rose-400 border-rose-500/20",
    };

    return (
        <span className={`inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-medium ${styles[status] || styles.COMPLETED}`}>
            {status}
        </span>
    );
}

export function SeverityBadge({ severity }: { severity: VulnSeverity }) {
    const styles: Record<VulnSeverity, string> = {
        CRITICAL: "bg-red-500/15 text-red-400 border-red-500/30",
        HIGH: "bg-orange-500/15 text-orange-400 border-orange-500/30",
        MEDIUM: "bg-amber-500/15 text-amber-400 border-amber-500/30",
        LOW: "bg-blue-500/15 text-blue-400 border-blue-500/30",
        INFO: "bg-slate-500/15 text-slate-400 border-slate-500/30",
    };

    return (
        <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-semibold uppercase tracking-wider ${styles[severity] || styles.INFO}`}>
            {severity}
        </span>
    );
}