import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import {cx} from "@/ui/cx";

const buttonVariants = cva(
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-sm font-semibold tracking-tight transition-colors focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
    {
        variants: {
            variant: {
                default:
                    "bg-primary text-primary-foreground hover:bg-primary/90",
                destructive:
                    "bg-destructive text-destructive-foreground hover:bg-destructive/90",
                outline:
                    "border border-input bg-transparent hover:bg-accent hover:text-accent-foreground text-foreground",
                secondary:
                    "bg-secondary text-secondary-foreground hover:bg-secondary/80",
                ghost: "hover:bg-accent hover:text-accent-foreground text-foreground",
                clear: "bg-transparent text-foreground hover:bg-accent/70",
                clear2: "bg-transparent text-muted-foreground hover:text-foreground hover:bg-accent/50",
                link: "text-primary underline-offset-4 hover:underline",
                neomorphic: "bg-card text-card-foreground border border-border shadow-[inset_0_1px_0_hsl(var(--foreground)/0.08),0_8px_20px_hsl(var(--background)/0.45)] hover:shadow-[inset_0_1px_0_hsl(var(--foreground)/0.12),0_10px_24px_hsl(var(--background)/0.5)]",
            },
            size: {
                default: "h-11 px-4 py-2.5 min-h-[44px] [&_svg]:size-4",
                sm: "h-9 px-3 text-xs min-h-[36px] [&_svg]:size-3.5",
                lg: "h-12 px-6 text-base min-h-[48px] [&_svg]:size-5",
                icon: "h-11 w-11 min-h-[44px] min-w-[44px] [&_svg]:size-5",
                freeflow: "h-auto min-h-0 px-0 py-0 text-inherit",
            },
        },
        defaultVariants: {
            variant: "default",
            size: "default",
        },
    },
);

export interface ButtonProps
    extends React.ButtonHTMLAttributes<HTMLButtonElement>,
        VariantProps<typeof buttonVariants> {
    asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    ({ className, variant, size, asChild = false, ...props }, ref) => {
        const Comp = asChild ? Slot : "button";
        return (
            <Comp
                className={cx(buttonVariants({ variant, size, className }))}
                ref={ref}
                {...props}
            />
        );
    },
);
Button.displayName = "Button";

export { Button, buttonVariants };
