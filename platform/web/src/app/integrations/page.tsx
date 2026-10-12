// src/app/integrations/page.tsx
import { Plug, Github, Webhook, Database, Cloud, CheckCircle2, AlertTriangle, Settings, ExternalLink } from "lucide-react";

export default function IntegrationsPage() {
    const integrations = [
        {
            name: "GitHub App",
            description: "Repository access, webhook events, PR check runs & annotations",
            icon: Github,
            status: "connected",
            detail: "lucid-security org • 12 repos • Webhooks active",
        },
        {
            name: "PostgreSQL",
            description: "Primary scan results store and vulnerability persistence layer",
            icon: Database,
            status: "connected",
            detail: "Port 5433 • lucid_ci database • 10 pool connections",
        },
        {
            name: "AWS SQS",
            description: "Async scan job queue for worker pool ingestion pipeline",
            icon: Cloud,
            status: "connected",
            detail: "us-east-1 • lucid-scan-queue • Polling active",
        },
        {
            name: "Webhook Endpoint",
            description: "Inbound GitHub webhook receiver for PR & push events",
            icon: Webhook,
            status: "warning",
            detail: "HMAC validation pending configuration",
        },
    ];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-headline-xl font-bold text-on-surface">Integrations</h1>
                <p className="mt-1 text-body-md text-on-surface-variant">
                    Connected services and infrastructure configuration.
                </p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                {integrations.map((integ) => {
                    const Icon = integ.icon;
                    return (
                        <div key={integ.name} className="rounded-lg border border-outline-variant/30 bg-surface-container-low p-5 hover:bg-surface-container transition">
                            <div className="flex items-start justify-between">
                                <div className="flex items-start gap-3">
                                    <div className="h-10 w-10 rounded-lg bg-surface-container flex items-center justify-center">
                                        <Icon className="h-5 w-5 text-primary" />
                                    </div>
                                    <div>
                                        <h3 className="text-body-lg font-semibold text-on-surface">{integ.name}</h3>
                                        <p className="mt-0.5 text-body-sm text-on-surface-variant">{integ.description}</p>
                                        <p className="mt-2 text-label-sm text-on-surface-variant/60 font-mono">{integ.detail}</p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 flex-shrink-0">
                                    {integ.status === "connected" ? (
                                        <span className="inline-flex items-center gap-1 rounded border border-secondary/30 bg-secondary/10 px-2 py-0.5 text-label-sm text-secondary font-medium">
                                            <CheckCircle2 className="h-3 w-3" />
                                            Connected
                                        </span>
                                    ) : (
                                        <span className="inline-flex items-center gap-1 rounded border border-warning/30 bg-warning/10 px-2 py-0.5 text-label-sm text-warning font-medium">
                                            <AlertTriangle className="h-3 w-3" />
                                            Warning
                                        </span>
                                    )}
                                    <button className="rounded border border-outline-variant/30 bg-surface-container p-1.5 hover:bg-surface-container-high transition">
                                        <Settings className="h-3.5 w-3.5 text-on-surface-variant" />
                                    </button>
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
