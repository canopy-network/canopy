import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cx } from "@/ui/cx";

const inputVariants = cva(
  "file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground flex w-full min-w-0 rounded-md border bg-transparent text-foreground transition-[color,box-shadow,border-color] outline-none file:inline-flex file:border-0 file:bg-transparent file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:ring-destructive/20 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "border-input/85 bg-secondary/70 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] hover:border-primary/35 focus-visible:border-primary focus-visible:ring-primary/25 focus-visible:ring-[3px]",
        wallet:
          "border-none bg-transparent focus-visible:outline-none focus-visible:ring-0",
        ghost:
          "border-transparent bg-transparent hover:border-input/85 hover:bg-secondary/40 focus-visible:border-primary focus-visible:ring-primary/25 focus-visible:ring-[3px]",
        destructive:
          "border-destructive/80 bg-destructive/10 focus-visible:border-destructive focus-visible:ring-destructive/35 focus-visible:ring-[3px]",
      },
      size: {
        sm: "h-8 px-2 py-1 text-sm file:h-6 file:text-xs",
        default: "h-9 px-3 py-1 text-base md:text-sm file:h-7 file:text-sm",
        lg: "h-10 px-4 py-2 text-lg file:h-8 file:text-base",
        wallet: "h-14 px-4 text-3xl file:h-8 file:text-base",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

type InputProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> &
  VariantProps<typeof inputVariants>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, variant, size, ...props }, ref) => {
    return (
      <input
        type={type}
        data-slot="input"
        className={cx(inputVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  },
);

Input.displayName = "Input";

export { Input };
