import { type VariantProps, cva } from "class-variance-authority";
import {cx} from "@/ui/cx";

const badgeVariants = cva(
    "inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium leading-none transition-colors focus:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
    {
        variants: {
            variant: {
                default:
                    "border-transparent bg-primary text-primary-foreground shadow-sm hover:bg-primary/90",
                secondary:
                    "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/90",
                destructive:
                    "border-transparent bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90",
                outline: "border-border text-foreground",
                virtual_active: "border-purple-500/30 bg-purple-500/15 text-purple-300",
                pending_launch: "border-yellow-500/30 bg-yellow-500/15 text-yellow-300",
                draft: "border-border/60 bg-muted/50 text-muted-foreground",
                rejected: "border-destructive/40 bg-destructive/15 text-red-300",
                graduated: "border-green-500/30 bg-green-500/15 text-green-300",
                failed: "border-destructive/40 bg-destructive/15 text-red-300",
                moderated: "border-orange-500/30 bg-orange-500/15 text-orange-300",
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
        <div className={cx(badgeVariants({ variant }), className)} {...props} />
    );
}

export { Badge, badgeVariants };
