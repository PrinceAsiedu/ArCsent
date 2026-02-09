import { Link, useLocation } from "wouter";
import {
    LayoutDashboard,
    ScanLine,
    ShieldAlert,
    Settings as SettingsIcon,
    LogOut,
} from "lucide-react";
import { cn } from "./lib/utils";
import { api } from "./lib/api";

export function Layout({ children }: { children: React.ReactNode }) {
    const [location] = useLocation();

    const handleLogout = () => {
        api.setToken("");
        window.location.reload();
    };

    const navItems = [
        { href: "/", label: "Dashboard", icon: LayoutDashboard },
        { href: "/scanners", label: "Scanners", icon: ScanLine },
        { href: "/findings", label: "Findings", icon: ShieldAlert },
        { href: "/settings", label: "Settings", icon: SettingsIcon },
    ];

    return (
        <div className="flex min-h-screen bg-[var(--paper)]">
            {/* Sidebar */}
            <aside className="w-64 border-r border-[var(--card-border)] bg-[var(--card)] px-4 py-6 flex flex-col">
                <div className="flex items-center gap-2 px-2 mb-8">
                    <div className="w-8 h-8 rounded-full bg-[var(--accent)] flex items-center justify-center text-white font-bold">
                        A
                    </div>
                    <span className="font-bold text-lg tracking-tight">ArCsent</span>
                </div>

                <nav className="flex-1 space-y-1">
                    {navItems.map((item) => {
                        const Icon = item.icon;
                        const isActive = location === item.href;
                        return (
                            <Link key={item.href} href={item.href}>
                                <a
                                    className={cn(
                                        "flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                                        isActive
                                            ? "bg-[var(--accent)]/10 text-[var(--accent)]"
                                            : "text-[var(--muted)] hover:bg-[var(--muted)]/5 hover:text-[var(--ink)]"
                                    )}
                                >
                                    <Icon size={18} />
                                    {item.label}
                                </a>
                            </Link>
                        );
                    })}
                </nav>

                <button
                    onClick={handleLogout}
                    className="flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-[var(--muted)] hover:bg-[var(--danger)]/10 hover:text-[var(--danger)] transition-colors mt-auto"
                >
                    <LogOut size={18} />
                    Logout
                </button>
            </aside>

            {/* Main Content */}
            <main className="flex-1 overflow-auto">
                <div className="container py-8">{children}</div>
            </main>
        </div>
    );
}
