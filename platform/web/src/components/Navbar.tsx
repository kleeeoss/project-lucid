import Link from "next/link";
import { Shield, Activity, GitPullRequest, Database } from "lucide-react";

export function Navbar() {
    return (
        <header className="sticky top-0 z-50 w-full border-b border-slate-800 bg-[#090d16]/80 backdrop-blur-md">
            <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
                <div className="flex items-center gap-8">
                    <Link href="/" className="flex items-center gap-2.5 font-bold tracking-tight text-white">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-sky-500/10 border border-sky-500/30 text-sky-400">
                            <Shield className="h-5 w-5" />
                        </div>
                        <div className="flex flex-col">
                            <span className="text-base font-semibold leading-none">Lucid-CI</span>
                            <span className="text-[10px] text-slate-400 leading-tight">AI DevSecOps Gateway</span>
                        </div>
                    </Link>
                    <nav className="hidden md:flex items-center gap-6 text-sm">
                        <Link href="/" className="flex items-center gap-1.5 text-slate-300 hover:text-white transition">
                            <Activity className="h-4 w-4 text-sky-400" />
                            Scans
                        </Link>
                        <Link href="/" className="flex items-center gap-1.5 text-slate-400 hover:text-slate-200 transition">
                            <GitPullRequest className="h-4 w-4" />
                            Pull Requests
                        </Link>
                    </nav>
                </div>

                <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-3 py-1 text-xs text-emerald-400">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                        Engine Online
                    </div>
                    <div className="flex items-center gap-2 rounded-full border border-slate-800 bg-slate-900 px-3 py-1 text-xs text-slate-400">
                        <Database className="h-3.5 w-3.5 text-slate-400" />
                        Port 5433
                    </div>
                </div>
            </div>
        </header>
    );
}