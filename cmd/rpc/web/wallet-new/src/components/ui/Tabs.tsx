"use client";

import * as React from "react";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { cva, type VariantProps } from "class-variance-authority";

import { cx } from "@/ui/cx";

const tabsListVariants = cva("inline-flex items-center justify-center", {
  variants: {
    variant: {
      default:
        "h-9 w-fit rounded-lg border border-border/70 bg-secondary/75 p-[3px] text-muted-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]",
      clear: "bg-transparent text-foreground h-auto w-full gap-2",
      outline: "h-auto w-full gap-2 bg-transparent text-foreground",
      wallet:
        "h-auto w-full justify-start gap-1 overflow-x-auto rounded-none border-b border-border/70 bg-transparent p-0 text-foreground flex-nowrap",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

const tabsTriggerVariants = cva(
  "inline-flex items-center justify-center text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:ring-[3px] focus-visible:outline-1 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      size: {
        default: "px-3 py-2",
        sm: "px-3 !py-1",
        lg: "px-4 py-3",
      },
      variant: {
        default:
          "h-[calc(100%-1px)] flex-1 rounded-md border border-transparent text-foreground focus-visible:border-ring focus-visible:ring-ring/40 focus-visible:outline-none data-[state=active]:border-border/90 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-[inset_0_1px_0_rgba(255,255,255,0.05),0_0_0_1px_rgba(53,205,72,0.16)]",
        clear:
          "rounded-lg border border-transparent bg-transparent text-muted-foreground leading-none tracking-normal data-[state=active]:border-border/80 data-[state=active]:bg-secondary/70 data-[state=active]:text-foreground",
        outline:
          "rounded-lg border border-border/80 bg-muted/65 text-muted-foreground leading-none tracking-normal data-[state=active]:border-primary/42 data-[state=active]:text-primary data-[state=active]:shadow-[0_0_0_1px_rgba(53,205,72,0.18)]",
        wallet:
          "shrink-0 -mb-[1px] rounded-none border-b-[2px] border-transparent bg-transparent px-4 py-3 pb-3 font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-primary",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface TabsListProps
  extends React.ComponentProps<typeof TabsPrimitive.List>,
    VariantProps<typeof tabsListVariants> {}

export interface TabsTriggerProps
  extends React.ComponentProps<typeof TabsPrimitive.Trigger>,
    VariantProps<typeof tabsTriggerVariants> {}

function Tabs({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cx("flex flex-col", className)}
      {...props}
    />
  );
}

function TabsList({ className, variant, ...props }: TabsListProps) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cx(tabsListVariants({ variant }), className)}
      {...props}
    />
  );
}

function TabsTrigger({ className, variant, size, ...props }: TabsTriggerProps) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cx(tabsTriggerVariants({ variant, size }), className)}
      {...props}
    />
  );
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cx("flex-1 outline-none", className)}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
