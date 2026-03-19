import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cx } from "@/ui/cx";

const cardVariants = cva("relative flex flex-col border shadow-sm", {
    variants: {
        variant: {
            default: "canopy-card-soft gap-5 text-card-foreground",
            dark: "surface-card gap-5 text-white",
            glass:
                "gap-5 border border-white/15 bg-white/[0.045] text-white backdrop-blur-xl shadow-[inset_0_1px_0_rgba(255,255,255,0.08),0_14px_28px_rgba(0,0,0,0.32)]",
            outline:
                "gap-5 border-border/85 bg-secondary/55 text-foreground backdrop-blur-md shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] hover:border-primary/35",
            ghost: "bg-transparent border-transparent shadow-none",
            gradient:
                "gap-5 border-primary/20 bg-[linear-gradient(145deg,hsl(0_0%_16%_/_0.95),hsl(0_0%_10%_/_0.95))] text-white shadow-[0_0_0_1px_hsl(var(--primary)/0.14),0_20px_36px_hsl(0_0%_0%/0.32)]",
            launchpad: "gap-4 lg:gap-5",
        },
        size: {
            default: "py-4 lg:py-6",
            launchpad: "py-0",
            sm: "py-4",
            lg: "py-8",
            xl: "py-12",
            none: "py-0",
        },
        padding: {
            default: "px-5 lg:px-6",
            launchpad: "px-0",
            sm: "px-4",
            lg: "px-8",
            xl: "px-12",
            none: "p-0",
            explorer: "px-6",
        },
        rounded: {
            default: "rounded-xl",
            lg: "rounded-lg",
        },
    },
    defaultVariants: {
        variant: "default",
        size: "default",
        padding: "default",
        rounded: "default",
    },
});

const Card = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof cardVariants>
>(({ className, variant, size, padding, rounded, ...props }, ref) => (
    <div
        ref={ref}
        data-slot="card"
        className={cx(
            cardVariants({ variant, size, padding, rounded }),
            variant !== "ghost" && "transition-all duration-200",
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
        data-slot="card-header"
        className={cx(
          "@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 px-6 has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-6",
          className,
        )}
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
        data-slot="card-title"
        className={cx("leading-none font-semibold", className)}
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
        data-slot="card-description"
        className={cx("text-muted-foreground text-sm", className)}
        {...props}
    />
));
CardDescription.displayName = "CardDescription";

const CardContent = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div ref={ref} data-slot="card-content" className={cx("px-6", className)} {...props} />
));
CardContent.displayName = "CardContent";

const CardFooter = React.forwardRef<
    HTMLDivElement,
    React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
    <div
        ref={ref}
        data-slot="card-footer"
        className={cx("flex items-center px-6 [.border-t]:pt-6", className)}
        {...props}
    />
));
CardFooter.displayName = "CardFooter";

const CardAction = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    data-slot="card-action"
    className={cx("col-start-2 row-span-2 row-start-1 self-start justify-self-end", className)}
    {...props}
  />
));
CardAction.displayName = "CardAction";

export {
    Card,
    CardHeader,
    CardFooter,
    CardTitle,
    CardAction,
    CardDescription,
    CardContent,
    cardVariants,
};
