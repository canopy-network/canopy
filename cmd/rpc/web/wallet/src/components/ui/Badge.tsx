import { type VariantProps, cva } from "class-variance-authority";
import { cx } from "@/ui/cx";

const badgeVariants = cva(
    "inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border px-2.5 py-0.5 text-[11px] font-semibold uppercase tracking-[0.02em] whitespace-nowrap [&>svg]:size-3 focus-visible:ring-ring/50 focus-visible:ring-[3px] transition-colors",
    {
        variants: {
            variant: {
                default: "border-primary/35 bg-primary/18 text-primary",
                secondary: "border-border/70 bg-secondary/85 text-secondary-foreground",
                destructive: "border-destructive/45 bg-destructive/18 text-red-200",
                outline: "border-border/85 bg-secondary/55 text-foreground",
                virtual_active: "border-primary/32 bg-primary/16 text-primary",
                pending_launch: "border-amber-500/35 bg-amber-500/15 text-amber-300",
                draft: "border-border/60 bg-muted/65 text-muted-foreground",
                rejected: "border-destructive/45 bg-destructive/18 text-red-200",
                graduated: "border-emerald-500/36 bg-emerald-500/16 text-emerald-300",
                failed: "border-destructive/45 bg-destructive/18 text-red-200",
                moderated: "border-orange-500/35 bg-orange-500/16 text-orange-300",
            },
        },
        defaultVariants: {
            variant: "default",
        },
    },
);

export interface BadgeProps
    extends React.HTMLAttributes<HTMLDivElement>,
        VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
    return (
        <div data-slot="badge" className={cx(badgeVariants({ variant }), className)} {...props} />
    );
}

export { Badge, badgeVariants };
