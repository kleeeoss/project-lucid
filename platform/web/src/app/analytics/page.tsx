// src/app/analytics/page.tsx
import {
    BarChart3,
    TrendingUp,
    TrendingDown,
    Clock,
    ShieldAlert,
    CheckCircle2,
    GitPullRequest,
} from "lucide-react";

export default function AnalyticsPage() {
    const weeklyData = [
        { week: "W36", scans: 142, findings: 18, mttf: 23 },
        { week: "W37", scans: 158, findings: 14, mttf: 19 },
        { week: "W38", scans: 171, findings: 22, mttf: 21 },
        { week: "W39", scans: 186, findings: 12, mttf: 16 },
    ];
    const maxScans = Math.max(...weeklyData.map(d => d.scans));

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-headline-xl font-bold text-on-surface">Analytics</h1>
                <p className="mt-1 text-body-md text-on-surface-variant">
                    Security trends, scan performance metrics, and team velocity insights.
                </p>
            </div>

            {/* Trend cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                {[
                    { label: "Scans / Week", value: "186", trend: "+12%", positive: true, icon: BarChart3, color: "text-primary" },
                    { label: "Mean Time to Fix", value: "16min", trend: "-18%", positive: true, icon: Clock, color: "text-secondary" },
                    { label: "New Findings", value: "12", trend: "-33%", positive: true, icon: ShieldAlert, color: "text-warning" },
                    { label: "Gate Pass Rate", value: "94%", trend: "+5%", positive: true, icon: CheckCircle2, color: "text-secondary" },
                ].map((metric) => {
                    const Icon = metric.icon;
                    return (
                        <div key={metric.label} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-4">
                            <div className="flex items-center justify-between">
                                <span className="text-label-sm text-on-surface-variant uppercase tracking-wider">{metric.label}</span>
                                <Icon className={`h-4 w-4 ${metric.color}`} />
                            </div>
                            <p className="mt-3 text-3xl font-bold text-on-surface">{metric.value}</p>
                            <div className="mt-1 flex items-center gap-1 text-label-sm">
                                {metric.positive ? (
                                    <TrendingUp className="h-3 w-3 text-secondary" />
                                ) : (
                                    <TrendingDown className="h-3 w-3 text-error" />
                                )}
                                <span className={metric.positive ? "text-secondary" : "text-error"}>
                                    {metric.trend}
                                </span>
                                <span className="text-on-surface-variant/50">vs last week</span>
                            </div>
                        </div>
                    );
                })}
            </div>

            {/* Weekly scan volume chart */}
            <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-5">
                <h2 className="text-headline-md font-semibold text-on-surface mb-4">Weekly Scan Volume</h2>
                <div className="flex items-end gap-6 h-40">
                    {weeklyData.map((week) => (
                        <div key={week.week} className="flex-1 flex flex-col items-center gap-2">
                            <span className="text-label-sm text-on-surface font-medium">{week.scans}</span>
                            <div className="w-full rounded-t-md bg-primary/30 transition-all" style={{ height: `${(week.scans / maxScans) * 100}%` }}>
                                <div className="w-full h-full rounded-t-md bg-gradient-to-t from-primary/60 to-primary" />
                            </div>
                            <span className="text-label-sm text-on-surface-variant">{week.week}</span>
                        </div>
                    ))}
                </div>
            </div>

            {/* Findings trend + MTTF */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-5">
                    <h2 className="text-headline-md font-semibold text-on-surface mb-4">Findings Trend</h2>
                    <div className="space-y-3">
                        {weeklyData.map((week) => (
                            <div key={week.week} className="flex items-center gap-3">
                                <span className="text-label-sm text-on-surface-variant w-10">{week.week}</span>
                                <div className="flex-1 h-3 rounded-full bg-surface-container overflow-hidden">
                                    <div
                                        className="h-full rounded-full bg-gradient-to-r from-error to-warning"
                                        style={{ width: `${(week.findings / 25) * 100}%` }}
                                    />
                                </div>
                                <span className="text-label-sm text-on-surface font-medium w-8 text-right">{week.findings}</span>
                            </div>
                        ))}
                    </div>
                </div>

                <div className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-5">
                    <h2 className="text-headline-md font-semibold text-on-surface mb-4">Mean Time to Fix (minutes)</h2>
                    <div className="space-y-3">
                        {weeklyData.map((week) => (
                            <div key={week.week} className="flex items-center gap-3">
                                <span className="text-label-sm text-on-surface-variant w-10">{week.week}</span>
                                <div className="flex-1 h-3 rounded-full bg-surface-container overflow-hidden">
                                    <div
                                        className="h-full rounded-full bg-gradient-to-r from-secondary/80 to-secondary"
                                        style={{ width: `${(week.mttf / 30) * 100}%` }}
                                    />
                                </div>
                                <span className="text-label-sm text-on-surface font-medium w-8 text-right">{week.mttf}m</span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
}
