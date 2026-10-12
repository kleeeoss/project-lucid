// src/app/settings/page.tsx
import { Settings, Shield, Bell, Code2, GitFork, Database, Key, Globe, ToggleRight } from "lucide-react";

export default function SettingsPage() {
    const settingSections = [
        {
            title: "General",
            icon: Settings,
            settings: [
                { label: "Organization Name", value: "lucid-security", type: "text" },
                { label: "Default Branch Protection", value: true, type: "toggle" },
                { label: "Auto-Scan on PR Open", value: true, type: "toggle" },
            ],
        },
        {
            title: "Security Policy",
            icon: Shield,
            settings: [
                { label: "Block Merge on Critical Findings", value: true, type: "toggle" },
                { label: "Require Sandbox Verification", value: true, type: "toggle" },
                { label: "Minimum Confidence Threshold", value: "75%", type: "text" },
                { label: "Auto-Apply Low-Risk Patches", value: false, type: "toggle" },
            ],
        },
        {
            title: "Notifications",
            icon: Bell,
            settings: [
                { label: "Email on Critical Finding", value: true, type: "toggle" },
                { label: "Slack Integration", value: false, type: "toggle" },
                { label: "Daily Digest Report", value: true, type: "toggle" },
            ],
        },
        {
            title: "AI Engine",
            icon: Code2,
            settings: [
                { label: "SLM Model Version", value: "v3.2-stable", type: "text" },
                { label: "Enable Auto-Remediation", value: true, type: "toggle" },
                { label: "Taint Analysis Depth", value: "5 hops", type: "text" },
                { label: "gVisor Sandbox Mode", value: true, type: "toggle" },
            ],
        },
    ];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-headline-xl font-bold text-on-surface">Settings</h1>
                <p className="mt-1 text-body-md text-on-surface-variant">
                    Workspace configuration, security policies, and engine preferences.
                </p>
            </div>

            <div className="space-y-4">
                {settingSections.map((section) => {
                    const SectionIcon = section.icon;
                    return (
                        <div key={section.title} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-5">
                            <div className="flex items-center gap-2 mb-4">
                                <SectionIcon className="h-4 w-4 text-primary" />
                                <h2 className="text-headline-md font-semibold text-on-surface">{section.title}</h2>
                            </div>
                            <div className="space-y-3">
                                {section.settings.map((setting) => (
                                    <div key={setting.label} className="flex items-center justify-between py-2 border-b border-outline-variant/10 last:border-b-0">
                                        <span className="text-body-md text-on-surface">{setting.label}</span>
                                        {setting.type === "toggle" ? (
                                            <div className={`relative w-10 h-5 rounded-full cursor-pointer transition ${setting.value ? "bg-primary" : "bg-surface-container-highest"}`}>
                                                <div className={`absolute top-0.5 h-4 w-4 rounded-full bg-on-surface transition-transform ${setting.value ? "translate-x-5" : "translate-x-0.5"}`} />
                                            </div>
                                        ) : (
                                            <span className="text-body-sm text-on-surface-variant font-mono bg-surface-container px-2.5 py-1 rounded border border-outline-variant/30">
                                                {String(setting.value)}
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    );
                })}
            </div>

            <div className="flex justify-end gap-3">
                <button className="rounded-md border border-outline-variant/30 bg-surface-container px-4 py-2 text-body-sm text-on-surface-variant hover:bg-surface-container-high transition">
                    Reset Defaults
                </button>
                <button className="rounded-md bg-primary-container px-4 py-2 text-body-sm font-semibold text-on-primary hover:bg-primary-container/90 transition shadow-lg shadow-primary-container/20">
                    Save Changes
                </button>
            </div>
        </div>
    );
}
