import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cx } from "@/ui/cx";

const buttonVariants = cva(
  "inline-flex items-center justify-center cursor-pointer gap-2 whitespace-nowrap rounded-[10px] text-sm font-semibold font-body transition-all duration-200 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] focus-visible:outline-none aria-invalid:ring-destructive/20 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-[0_0_0_1px_hsl(var(--primary)/0.35)_inset,0_8px_18px_hsl(var(--primary)/0.22)] hover:bg-primary/90 hover:-translate-y-[1px] btn-glow",
        destructive:
          "bg-destructive text-white shadow-[0_0_0_1px_rgba(239,68,68,0.35)_inset,0_8px_16px_rgba(239,68,68,0.2)] hover:bg-destructive/90 hover:-translate-y-[1px] focus-visible:ring-destructive/25",
        outline:
          "border border-border/85 bg-secondary/70 text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.05)] hover:bg-secondary/85 hover:border-primary/40",
        secondary:
          "border border-border/70 bg-secondary text-secondary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] hover:bg-secondary/85 hover:border-primary/30",
        ghost: "text-foreground hover:bg-accent/70 hover:text-accent-foreground",
        clear: "border-none bg-transparent hover:bg-accent/70 hover:text-accent-foreground",
        clear2: "bg-transparent hover:bg-accent/70 hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
        neomorphic:
          "border border-white/15 bg-[linear-gradient(145deg,rgba(255,255,255,0.14),rgba(255,255,255,0.06))] text-white hover:border-primary/35",
      },
      size: {
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        sm: "h-8 rounded-md gap-1.5 px-3 has-[>svg]:px-2.5",
        lg: "h-10 rounded-md px-6 has-[>svg]:px-4",
        icon: "size-9",
        freeflow: "",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ComponentProps<"button">,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";

    return (
      <Comp
        ref={ref}
        data-slot="button"
        className={cx(buttonVariants({ variant, size, className }))}
        {...props}
      />
    );
  },
);

Button.displayName = "Button";

export { Button, buttonVariants };
