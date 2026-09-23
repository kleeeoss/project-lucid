import { ReactNode } from "react";

interface MetricCardProps {
    title: string;
    value: string | number;
    subtitle: string;
    icon: ReactNode;
    trend?: string;
    isPositive?: boolean;
}

export function MetricCard({ title, value, subtitle, icon, trend }: MetricCardProps) {
    return (
        <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-5 backdrop-blur-sm">
            <div className="flex items-center justify-between">
                <span className="text-xs font-medium uppercase tracking-wider text-slate-400">{title}</span>
                <div className="rounded-lg border border-slate-800 bg-slate-800/50 p-2 text-slate-300">
                    {icon}
                </div>
            </div>
            <div className="mt-4 flex items-baseline gap-2">
                <span className="text-3xl font-bold tracking-tight text-white">{value}</span>
                {trend && (
                    <span className="text-xs font-medium text-emerald-400">{trend}</span>
                )}
            </div>
            <p className="mt-1 text-xs text-slate-400">{subtitle}</p>
        </div>
    );
}