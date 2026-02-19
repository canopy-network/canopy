import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import {cx} from "@/ui/cx";

const cardVariants = cva("border text-card-foreground shadow-sm", {
    variants: {
        variant: {
            default: "bg-card border-border",
            dark: "bg-background border-border/70",
            glass: "bg-card/70 border-border/70 backdrop-blur-md",
            outline: "bg-transparent border-border",
            ghost: "bg-transparent border-transparent shadow-none",
            gradient: "bg-gradient-to-br from-card via-card to-accent/50 border-border",
            launchpad: "bg-card border-border shadow-[0_0_0_1px_hsl(var(--foreground)/0.04),0_18px_40px_hsl(var(--background)/0.55)]",
        },
        size: {
            default: "min-h-0",
            launchpad: "min-h-[18rem]",
            sm: "min-h-[8rem]",
            lg: "min-h-[14rem]",
            xl: "min-h-[20rem]",
            none: "min-h-0",
        },
        padding: {
            default: "p-6",
            launchpad: "p-7",
            sm: "p-3",
            lg: "p-8",
            xl: "p-10",
            none: "p-0",
            explorer: "p-4 sm:p-5 lg:p-6",
        },
        rounded: {
            default: "rounded-xl",
            lg: "rounded-2xl",
        },
    },
    defaultVariants: {
        variant: "default",
        size: "default",
        padding: "none",
        rounded: "default",
    },
});

const Card = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof cardVariants>
>(({ className, variant, size, padding, rounded, ...props }, ref) => (
    <div
        ref={ref}
        className={cx(
            cardVariants({ variant, size, padding, rounded }),
            className,
        )}
        {...props}
    />
));
Card.displayName = "Card";

const CardHeader = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div
        ref={ref}
        className={cx("flex flex-col space-y-1.5 p-6", className)}
        {...props}
    />
));
CardHeader.displayName = "CardHeader";

const CardTitle = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div
        ref={ref}
        className={cx("font-semibold leading-none tracking-tight", className)}
        {...props}
    />
));
CardTitle.displayName = "CardTitle";

const CardDescription = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div
        ref={ref}
        className={cx("text-sm text-muted-foreground", className)}
        {...props}
    />
));
CardDescription.displayName = "CardDescription";

const CardContent = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div ref={ref} className={cx("p-6 pt-0", className)} {...props} />
));
CardContent.displayName = "CardContent";

const CardFooter = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div
        ref={ref}
        className={cx("flex items-center p-6 pt-0", className)}
        {...props}
    />
));
CardFooter.displayName = "CardFooter";

export {
    Card,
    CardHeader,
    CardFooter,
    CardTitle,
    CardDescription,
    CardContent,
    cardVariants,
};
