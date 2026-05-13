import { type VariantProps, cva } from "class-variance-authority";
import { cx } from "@/ui/cx";
import { WALLET_BADGE_CLASS } from "@/components/ui/badgeStyles";

const badgeVariants = cva(
    `${WALLET_BADGE_CLASS} w-fit shrink-0 gap-1 uppercase whitespace-nowrap [&>svg]:size-3 focus-visible:ring-ring/50 focus-visible:ring-[3px] transition-colors`,
    {
        variants: {
            variant: {
                default: "",
                secondary: "",
                destructive: "",
                outline: "",
                virtual_active: "",
                pending_launch: "",
                draft: "",
                rejected: "",
                graduated: "",
                failed: "",
                moderated: "",
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
