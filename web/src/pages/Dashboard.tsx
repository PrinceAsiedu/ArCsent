import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { api, type SignatureStatus, type SystemStatus } from "../lib/api";
import { Activity, Shield } from "lucide-react";

export default function Dashboard() {
    const [status, setStatus] = useState<SystemStatus | null>(null);
    const [signatureStatus, setSignatureStatus] = useState<SignatureStatus | null>(null);
    const [sigError, setSigError] = useState<string>("");
    const [sigLoading, setSigLoading] = useState<boolean>(false);
    const [metricsError, setMetricsError] = useState<string>("");
    const [metrics, setMetrics] = useState<Record<string, number>>({});
    const [metricsLoading, setMetricsLoading] = useState<boolean>(false);

    useEffect(() => {
        api.getStatus().then(setStatus).catch(console.error);
        refreshSignatures();
        refreshMetrics();
    }, []);

    const refreshSignatures = () => {
        setSigLoading(true);
        api.getSignaturesStatus()
            .then((data) => {
                setSignatureStatus(data);
                setSigError("");
            })
            .catch((err) => setSigError(err.message || "Failed to load signatures"))
            .finally(() => setSigLoading(false));
    };

    const handleSignatureUpdate = () => {
        setSigLoading(true);
        api.triggerSignaturesUpdate()
            .then((data) => {
                setSignatureStatus(data);
                setSigError("");
            })
            .catch((err) => setSigError(err.message || "Update failed"))
            .finally(() => setSigLoading(false));
    };

    const refreshMetrics = () => {
        setMetricsLoading(true);
        api.getMetricsText()
            .then((raw) => {
                const parsed: Record<string, number> = {};
                raw.split("\n").forEach((line) => {
                    const trimmed = line.trim();
                    if (!trimmed || trimmed.startsWith("#")) {
                        return;
                    }
                    const parts = trimmed.split(/\s+/);
                    if (parts.length < 2) {
                        return;
                    }
                    const name = parts[0];
                    const value = Number(parts[1]);
                    if (Number.isFinite(value) && name.startsWith("arcsent_")) {
                        parsed[name] = value;
                    }
                });
                setMetrics(parsed);
                setMetricsError("");
            })
            .catch((err) => setMetricsError(err.message || "Failed to load metrics"))
            .finally(() => setMetricsLoading(false));
    };

    const sources = signatureStatus?.sources || {};
    const sourceList = Object.values(sources);
    const totalSources = sourceList.length;
    const failedSources = sourceList.filter((src) => src.error).length;
    const okSources = totalSources - failedSources;

    const formatTime = (value?: string) => {
        if (!value) return "—";
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) return value;
        return parsed.toLocaleString();
    };

    return (
        <div className="space-y-6">
            <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">System Status</CardTitle>
                        <Activity className="h-4 w-4 text-[var(--muted)]" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold capitalize">
                            {status?.status || "Unknown"}
                        </div>
                        <p className="text-xs text-[var(--muted)]">
                            {status?.running_as_root ? "Running as root (Warning)" : "Secure mode"}
                        </p>
                    </CardContent>
                </Card>
                {/* Placeholders for other stats */}
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active Findings</CardTitle>
                        <Shield className="h-4 w-4 text-[var(--muted)]" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">--</div>
                        <p className="text-xs text-[var(--muted)]">Across all scanners</p>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Metrics</CardTitle>
                        <Button
                            size="sm"
                            variant="ghost"
                            isLoading={metricsLoading}
                            onClick={refreshMetrics}
                        >
                            Refresh
                        </Button>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Plugins:</span>{" "}
                            {metrics["arcsent_plugins_total"] ?? "—"}
                        </div>
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Jobs:</span>{" "}
                            {metrics["arcsent_jobs_total"] ?? "—"}
                        </div>
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Results:</span>{" "}
                            {metrics["arcsent_results_total"] ?? "—"}
                        </div>
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Findings:</span>{" "}
                            {metrics["arcsent_findings_total"] ?? "—"}
                        </div>
                        {metricsError && (
                            <div className="text-xs text-[var(--danger)]">{metricsError}</div>
                        )}
                    </CardContent>
                </Card>
                <Card className="md:col-span-2">
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Signatures</CardTitle>
                        <Button
                            size="sm"
                            variant="secondary"
                            isLoading={sigLoading}
                            onClick={handleSignatureUpdate}
                        >
                            Update now
                        </Button>
                    </CardHeader>
                    <CardContent className="space-y-2">
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Last run:</span>{" "}
                            {formatTime(signatureStatus?.last_run)}
                        </div>
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Next run:</span>{" "}
                            {formatTime(signatureStatus?.next_run)}
                        </div>
                        <div className="text-sm">
                            <span className="text-[var(--muted)]">Sources:</span>{" "}
                            {totalSources > 0 ? `${okSources} ok / ${failedSources} failed` : "—"}
                        </div>
                        {signatureStatus?.airgap_mode && (
                            <div className="text-xs text-[var(--muted)]">
                                Air-gapped import: {signatureStatus?.airgap_import_path || "enabled"}
                            </div>
                        )}
                        {sigError && <div className="text-xs text-[var(--danger)]">{sigError}</div>}
                        {!sigError && (
                            <button
                                onClick={refreshSignatures}
                                className="text-xs text-[var(--accent)] hover:underline"
                            >
                                Refresh status
                            </button>
                        )}
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
