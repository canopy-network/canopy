import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import {cx} from "@/ui/cx";

const buttonVariants = cva(
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-sm font-semibold tracking-tight transition-all duration-150 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/40 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 font-display",
    {
        variants: {
            variant: {
                default:
                    "bg-primary text-primary-foreground hover:bg-primary/90 shadow-glow-sm hover:shadow-glow btn-glow",
                destructive:
                    "bg-destructive text-destructive-foreground hover:bg-destructive/90",
                outline:
                    "border border-border/70 bg-transparent hover:bg-accent hover:border-primary/30 text-foreground",
                secondary:
                    "bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border/50",
                ghost: "hover:bg-accent/70 hover:text-foreground text-muted-foreground",
                clear: "bg-transparent text-foreground hover:bg-accent/60",
                clear2: "bg-transparent text-muted-foreground hover:text-foreground hover:bg-accent/50",
                link: "text-primary underline-offset-4 hover:underline p-0 h-auto",
                neomorphic: "bg-card text-card-foreground border border-border shadow-inner-top hover:border-primary/20 hover:shadow-glow-sm",
            },
            size: {
                default: "h-10 px-4 py-2 min-h-[40px] [&_svg]:size-4",
                sm: "h-8 px-3 text-xs min-h-[32px] [&_svg]:size-3.5",
                lg: "h-11 px-5 text-base min-h-[44px] [&_svg]:size-4.5",
                icon: "h-10 w-10 min-h-[40px] min-w-[40px] [&_svg]:size-4",
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
