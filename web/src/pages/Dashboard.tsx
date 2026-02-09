import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { api, type SignatureStatus, type SystemStatus } from "../lib/api";
import { Activity, Shield, Database, RefreshCw, CheckCircle, AlertCircle, HardDrive } from "lucide-react";
import { Badge } from "../components/Badge";

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
        api.getMetrics()
            .then((data) => {
                setMetrics(data);
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
        if (Number.isNaN(parsed.getTime()) || parsed.getTime() === 0) return "—";
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

                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active Findings</CardTitle>
                        <Shield className="h-4 w-4 text-[var(--muted)]" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {metrics["arcsent_findings_total"] ?? "—"}
                        </div>
                        <p className="text-xs text-[var(--muted)]">Total findings found</p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Scanners</CardTitle>
                        <HardDrive className="h-4 w-4 text-[var(--muted)]" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {metrics["arcsent_plugins_total"] ?? "—"}
                        </div>
                        <div className="text-xs text-[var(--muted)] flex justify-between">
                            <span>Jobs: {metrics["arcsent_jobs_total"] ?? 0}</span>
                            <span>Results: {metrics["arcsent_results_total"] ?? 0}</span>
                        </div>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Signatures</CardTitle>
                        <Database className="h-4 w-4 text-[var(--muted)]" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {okSources} <span className="text-sm font-normal text-[var(--muted)]">/ {totalSources} ok</span>
                        </div>
                        <p className="text-xs text-[var(--muted)]">
                            Last update: {formatTime(signatureStatus?.last_run)}
                        </p>
                    </CardContent>
                </Card>
            </div>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
                <Card className="col-span-4">
                    <CardHeader className="flex flex-row items-center justify-between">
                        <CardTitle>Signatures Status</CardTitle>
                        <Button
                            size="sm"
                            variant="secondary"
                            isLoading={sigLoading}
                            onClick={handleSignatureUpdate}
                        >
                            <RefreshCw className="mr-2 h-3 w-3" />
                            Update Now
                        </Button>
                    </CardHeader>
                    <CardContent>
                        {sourceList.length === 0 ? (
                            <div className="text-center py-6 text-[var(--muted)]">
                                No signature sources configured.
                            </div>
                        ) : (
                            <div className="space-y-4">
                                {sourceList.map((src) => (
                                    <div key={src.source} className="flex items-center justify-between border-b border-[var(--card-border)] pb-2 last:border-0 last:pb-0">
                                        <div className="flex items-center gap-3">
                                            {src.error ? (
                                                <AlertCircle className="h-5 w-5 text-[var(--danger)]" />
                                            ) : (
                                                <CheckCircle className="h-5 w-5 text-[var(--success)]" />
                                            )}
                                            <div>
                                                <div className="font-medium text-sm">{src.source}</div>
                                                <div className="text-xs text-[var(--muted)]">
                                                    {src.bytes ? `${(src.bytes / 1024).toFixed(1)} KB` : "0 KB"} • {formatTime(src.updated_at)}
                                                </div>
                                            </div>
                                        </div>
                                        {src.error && (
                                            <Badge variant="danger">Error</Badge>
                                        )}
                                        {!src.error && (
                                            <Badge variant="outline">Synced</Badge>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                        {sigError && (
                            <div className="mt-4 p-3 bg-[rgba(164,31,31,0.1)] text-[var(--danger)] text-sm rounded-md">
                                {sigError}
                            </div>
                        )}
                    </CardContent>
                </Card>
                <Card className="col-span-3">
                    <CardHeader>
                        <CardTitle>Metrics Overview</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-4">
                            <div className="flex justify-between items-center">
                                <span className="text-sm font-medium">Scans Completed</span>
                                <span className="text-sm font-bold text-[var(--accent)]">{metrics["arcsent_results_total"] ?? 0}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-sm font-medium">Findings Detected</span>
                                <span className="text-sm font-bold text-[var(--warning)]">{metrics["arcsent_findings_total"] ?? 0}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-sm font-medium">Active Plugins</span>
                                <span className="text-sm font-bold">{metrics["arcsent_plugins_total"] ?? 0}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-sm font-medium">Scheduled Jobs</span>
                                <span className="text-sm font-bold">{metrics["arcsent_jobs_total"] ?? 0}</span>
                            </div>
                        </div>
                        <div className="mt-6 flex justify-end">
                            <Button
                                size="sm"
                                variant="ghost"
                                isLoading={metricsLoading}
                                onClick={refreshMetrics}
                            >
                                <RefreshCw className="mr-2 h-3 w-3" />
                                Refresh Metrics
                            </Button>
                        </div>
                        {metricsError && (
                            <div className="mt-2 text-xs text-[var(--danger)]">{metricsError}</div>
                        )}
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
