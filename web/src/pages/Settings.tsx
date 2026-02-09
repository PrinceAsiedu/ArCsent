import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { api } from "../lib/api";

export default function Settings() {
    const [token, setToken] = useState("");
    const [status, setStatus] = useState<string>("");

    useEffect(() => {
        setToken(api.getToken() || "");
    }, []);

    const handleSave = () => {
        api.setToken(token);
        setStatus("Token saved successfully.");
        setTimeout(() => setStatus(""), 3000);
    };

    return (
        <div className="space-y-6">
            <h1 className="text-3xl font-bold tracking-tight">Settings</h1>

            <div className="grid gap-6 max-w-xl">
                <Card>
                    <CardHeader>
                        <CardTitle>Connection Settings</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="space-y-2">
                            <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                                API Token
                            </label>
                            <input
                                type="password"
                                value={token}
                                onChange={(e) => setToken(e.target.value)}
                                className="flex h-10 w-full rounded-md border border-[var(--card-border)] bg-transparent px-3 py-2 text-sm placeholder:text-[var(--muted)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:ring-offset-2"
                                placeholder="Enter your API token"
                            />
                            <p className="text-xs text-[var(--muted)]">
                                The token required to authenticate with the local ArCsent daemon.
                            </p>
                        </div>

                        <Button onClick={handleSave}>Save Configuration</Button>

                        {status && (
                            <p className="text-sm font-medium text-[var(--success)]">
                                {status}
                            </p>
                        )}
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
