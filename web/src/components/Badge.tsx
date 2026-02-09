import React from "react";
import { cn } from "../lib/utils";

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
    variant?: "default" | "success" | "warning" | "danger" | "outline";
}

export function Badge({ className, variant = "default", children, ...props }: BadgeProps) {
    const variants = {
        default: "bg-[var(--muted)]/10 text-[var(--muted-foreground)]",
        success: "bg-[var(--success)]/10 text-[var(--success)]",
        warning: "bg-[var(--warning)]/10 text-[var(--warning)]",
        danger: "bg-[var(--danger)]/10 text-[var(--danger)]",
        outline: "border border-[var(--card-border)] text-[var(--ink)]",
    };

    return (
        <span className={cn("badge", variants[variant], className)} {...props}>
            {children}
        </span>
    );
}
