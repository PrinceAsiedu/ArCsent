import { useEffect, useState } from "react";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "../components/Table";
import { Badge } from "../components/Badge";
import { api, type Finding } from "../lib/api";
import { format } from "date-fns";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";

export default function Findings() {
    const [findings, setFindings] = useState<Finding[]>([]);

    useEffect(() => {
        api.getFindings().then(setFindings).catch(console.error);
    }, []);

    const getSeverityBadge = (severity: string) => {
        const s = severity.toLowerCase();
        switch (s) {
            case "critical":
            case "high":
                return <Badge variant="danger">{severity}</Badge>;
            case "medium":
                return <Badge variant="warning">{severity}</Badge>;
            case "low":
                return <Badge variant="outline">{severity}</Badge>;
            default:
                return <Badge variant="default">{severity}</Badge>;
        }
    };

    return (
        <div className="space-y-6">
            <h1 className="text-3xl font-bold tracking-tight">Findings</h1>
            <Card>
                <CardHeader>
                    <CardTitle>Recent Activity</CardTitle>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Severity</TableHead>
                                <TableHead>Category</TableHead>
                                <TableHead>Scanner</TableHead>
                                <TableHead>Description</TableHead>
                                <TableHead>Time</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {findings.map((finding, i) => (
                                <TableRow key={i}>
                                    <TableCell>{getSeverityBadge(finding.severity)}</TableCell>
                                    <TableCell>
                                        <Badge variant="outline">{finding.category}</Badge>
                                    </TableCell>
                                    <TableCell>{finding.scanner_name}</TableCell>
                                    <TableCell>{finding.description}</TableCell>
                                    <TableCell>
                                        {finding.occurred_at
                                            ? format(new Date(finding.occurred_at), "PPpp")
                                            : "-"}
                                    </TableCell>
                                </TableRow>
                            ))}
                            {findings.length === 0 && (
                                <TableRow>
                                    <TableCell colSpan={5} className="text-center py-8 text-[var(--muted)]">
                                        No findings recorded.
                                    </TableCell>
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>
        </div>
    );
}
