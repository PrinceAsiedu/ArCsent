import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { Badge } from "../components/Badge";
import { api } from "../lib/api";
import { Play } from "lucide-react";

export default function Scanners() {
    const [scanners, setScanners] = useState<string[]>([]);
    const [loading, setLoading] = useState<Record<string, boolean>>({});

    useEffect(() => {
        api
            .getScanners()
            .then((data) => setScanners(data.plugins || []))
            .catch(console.error);
    }, []);

    const handleTrigger = async (plugin: string) => {
        setLoading((prev) => ({ ...prev, [plugin]: true }));
        try {
            await api.triggerScan(plugin);
            // Ideally show a toast here
        } catch (err) {
            console.error(err);
        } finally {
            setLoading((prev) => ({ ...prev, [plugin]: false }));
        }
    };

    return (
        <div className="space-y-6">
            <h1 className="text-3xl font-bold tracking-tight">Scanners</h1>
            <p className="text-[var(--muted)]">
                Manage and trigger security scanners.
            </p>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {scanners.map((plugin) => (
                    <Card key={plugin}>
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                            <CardTitle className="text-base font-medium">{plugin}</CardTitle>
                            <Badge variant="outline">Idle</Badge>
                        </CardHeader>
                        <CardContent>
                            <div className="pt-4">
                                <Button
                                    variant="secondary"
                                    size="sm"
                                    className="w-full"
                                    onClick={() => handleTrigger(plugin)}
                                    isLoading={loading[plugin]}
                                >
                                    {!loading[plugin] && <Play className="mr-2 h-3 w-3" />}
                                    Trigger Scan
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                ))}
                {scanners.length === 0 && (
                    <div className="col-span-full text-center text-[var(--muted)] py-12">
                        No scanners registered. Check your configuration.
                    </div>
                )}
            </div>
        </div>
    );
}
