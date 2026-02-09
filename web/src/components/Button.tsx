import React from "react";
import { cn } from "../lib/utils";
import { Loader2 } from "lucide-react";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: "primary" | "secondary" | "ghost" | "danger";
    size?: "sm" | "md" | "lg";
    isLoading?: boolean;
}

export function Button({
    className,
    variant = "primary",
    size = "md",
    isLoading = false,
    children,
    ...props
}: ButtonProps) {
    const variantClasses = {
        primary: "btn-primary",
        secondary: "btn-secondary",
        ghost: "btn-ghost",
        danger: "bg-[var(--danger)] text-white hover:opacity-90",
    };

    const sizeClasses = {
        sm: "h-8 px-3 text-xs",
        md: "h-10 px-4 py-2",
        lg: "h-12 px-8 text-lg",
    };

    return (
        <button
            className={cn(
                "btn",
                variantClasses[variant],
                sizeClasses[size],
                isLoading && "opacity-70 cursor-wait",
                className
            )}
            disabled={isLoading || props.disabled}
            {...props}
        >
            {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {children}
        </button>
    );
}
