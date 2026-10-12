// src/app/remediation/page.tsx
import {
    Wrench,
    CheckCircle2,
    Clock,
    AlertTriangle,
    ArrowRight,
    Sparkles,
    Play,
    RotateCcw,
} from "lucide-react";

export default function RemediationPage() {
    const remediations = [
        {
            id: "1",
            title: "SQL Injection in store.go:84",
            cwe: "CWE-89",
            severity: "CRITICAL",
            status: "ready",
            repo: "lucid-engine",
            pr: 442,
            patchPreview: "Use parameterized query: db.Query(\"SELECT ... WHERE id = $1\", id)",
            confidence: 97,
            estimatedImpact: "Eliminates SQL injection vector on /api/repos endpoint",
        },
        {
            id: "2",
            title: "HMAC Signature Bypass in auth.go:126",
            cwe: "CWE-347",
            severity: "HIGH",
            status: "ready",
            repo: "lucid-engine",
            pr: 441,
            patchPreview: "Add crypto/hmac.Equal signature check before processing webhook",
            confidence: 91,
            estimatedImpact: "Prevents forged webhook event injection",
        },
        {
            id: "3",
            title: "Slowloris DoS in monitor.go:52",
            cwe: "CWE-400",
            severity: "MEDIUM",
            status: "applied",
            repo: "lucid-platform",
            pr: 489,
            patchPreview: "Set http.Client.Timeout = 30 * time.Second",
            confidence: 78,
            estimatedImpact: "Prevents connection exhaustion under load",
        },
        {
            id: "4",
            title: "Verbose Error Disclosure in main.go:45",
            cwe: "CWE-209",
            severity: "LOW",
            status: "dismissed",
            repo: "lucid-engine",
            pr: 440,
            patchPreview: "Return generic error message, log details server-side",
            confidence: 65,
            estimatedImpact: "Prevents internal information leakage",
        },
    ];

    const statusConfig: Record<string, { label: string; color: string; icon: any }> = {
        ready: { label: "Ready to Apply", color: "text-primary bg-primary/10 border-primary/30", icon: Play },
        applied: { label: "Applied", color: "text-secondary bg-secondary/10 border-secondary/30", icon: CheckCircle2 },
        dismissed: { label: "Dismissed", color: "text-on-surface-variant bg-surface-container border-outline-variant/30", icon: RotateCcw },
    };

    const severityColors: Record<string, string> = {
        CRITICAL: "bg-error text-on-error",
        HIGH: "bg-orange-500/20 text-orange-400 border border-orange-500/30",
        MEDIUM: "bg-primary/15 text-primary border border-primary/30",
        LOW: "bg-surface-container text-on-surface-variant border border-outline-variant/30",
    };

    return (
        <div className="space-y-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-headline-xl font-bold text-on-surface">Remediation</h1>
                    <p className="mt-1 text-body-md text-on-surface-variant">
                        AI-generated patches and 1-click fixes for detected vulnerabilities.
                    </p>
                </div>
                <div className="flex items-center gap-2 text-label-sm text-secondary">
                    <Sparkles className="h-4 w-4" />
                    <span>SLM Patch Engine v3.2 Active</span>
                </div>
            </div>

            {/* Summary stats */}
            <div className="grid grid-cols-3 gap-3">
                <div className="rounded-lg border border-primary/30 bg-primary/10 p-4 text-center">
                    <p className="text-2xl font-bold text-primary">{remediations.filter(r => r.status === "ready").length}</p>
                    <p className="text-label-sm text-primary">Ready to Apply</p>
                </div>
                <div className="rounded-lg border border-secondary/30 bg-secondary/10 p-4 text-center">
                    <p className="text-2xl font-bold text-secondary">{remediations.filter(r => r.status === "applied").length}</p>
                    <p className="text-label-sm text-secondary">Applied</p>
                </div>
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container p-4 text-center">
                    <p className="text-2xl font-bold text-on-surface-variant">{remediations.filter(r => r.status === "dismissed").length}</p>
                    <p className="text-label-sm text-on-surface-variant">Dismissed</p>
                </div>
            </div>

            <div className="space-y-3">
                {remediations.map((rem) => {
                    const statusCfg = statusConfig[rem.status];
                    const StatusIcon = statusCfg.icon;

                    return (
                        <div key={rem.id} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-4 hover:bg-surface-container transition">
                            <div className="flex items-start justify-between gap-4">
                                <div className="flex-1">
                                    <div className="flex items-center gap-2 flex-wrap">
                                        <span className={`rounded px-1.5 py-0.5 text-label-sm font-bold ${severityColors[rem.severity]}`}>
                                            {rem.severity}
                                        </span>
                                        <h3 className="text-body-lg font-semibold text-on-surface">{rem.title}</h3>
                                        <code className="text-code-sm text-on-surface-variant font-mono bg-surface-container px-1.5 py-0.5 rounded">
                                            {rem.cwe}
                                        </code>
                                    </div>

                                    <div className="mt-2 rounded-md bg-surface-dim border border-outline-variant/20 px-3 py-2">
                                        <p className="text-code-sm font-mono text-on-surface-variant">
                                            <span className="text-secondary">Fix:</span> {rem.patchPreview}
                                        </p>
                                    </div>

                                    <div className="flex items-center gap-3 mt-2 text-label-sm text-on-surface-variant/60">
                                        <span>{rem.repo}</span>
                                        <span>•</span>
                                        <span>PR #{rem.pr}</span>
                                        <span>•</span>
                                        <span>Confidence: {rem.confidence}%</span>
                                        <span>•</span>
                                        <span>{rem.estimatedImpact}</span>
                                    </div>
                                </div>
                                <div className="flex flex-col items-end gap-2 flex-shrink-0">
                                    <span className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-label-sm font-medium ${statusCfg.color}`}>
                                        <StatusIcon className="h-3 w-3" />
                                        {statusCfg.label}
                                    </span>
                                    {rem.status === "ready" && (
                                        <button className="flex items-center gap-1 rounded bg-primary-container px-3 py-1.5 text-label-sm font-semibold text-on-primary hover:bg-primary-container/90 transition">
                                            Apply Patch <ArrowRight className="h-3 w-3" />
                                        </button>
                                    )}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
