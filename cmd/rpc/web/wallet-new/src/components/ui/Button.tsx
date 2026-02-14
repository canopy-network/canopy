import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import {cx} from "@/ui/cx";

const buttonVariants = cva(
    "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
    {
        variants: {
            variant: {
                default:
                    "bg-primary text-primary-foreground hover:bg-primary/90",
                destructive:
                    "bg-red-500/15 text-red-400 hover:bg-red-500/25",
                outline:
                    "border border-bg-accent bg-transparent hover:bg-bg-accent text-text-primary",
                secondary:
                    "bg-bg-tertiary text-text-primary hover:bg-bg-accent",
                ghost: "hover:bg-bg-accent text-text-primary",
                link: "text-primary underline-offset-4 hover:underline",
            },
            size: {
                default: "h-11 px-4 py-2.5 min-h-[44px] [&_svg]:size-4",
                sm: "h-9 px-3 text-xs min-h-[36px] [&_svg]:size-3.5",
                lg: "h-12 px-6 text-base min-h-[48px] [&_svg]:size-5",
                icon: "h-11 w-11 min-h-[44px] min-w-[44px] [&_svg]:size-5",
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
