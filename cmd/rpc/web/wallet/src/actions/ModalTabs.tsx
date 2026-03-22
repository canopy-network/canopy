import React from 'react';
import { LucideIcon } from "@/components/ui/LucideIcon";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/Tabs";

export interface Tab {
    value: string;
    label: string;
    icon?: string;
}

interface ModalTabsProps {
    tabs: Tab[];
    activeTab?: Tab;
    onTabChange?: (tab: Tab) => void;
}

export const ModalTabs: React.FC<ModalTabsProps> = ({
                                                        tabs,
                                                        activeTab,
                                                        onTabChange,
                                                    }) => {
    const activeValue = activeTab?.value ?? tabs[0]?.value ?? "";

    return (
        <div className="mb-3 sm:mb-4 md:mb-6 px-1 sm:px-2 md:px-3">
            <Tabs
                value={activeValue}
                onValueChange={(value) => {
                    const next = tabs.find((tab) => tab.value === value);
                    if (next) onTabChange?.(next);
                }}
            >
                <TabsList
                    variant="wallet"
                    className="w-full gap-1 sm:gap-2 overflow-x-auto whitespace-nowrap no-scrollbar [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
                >
                    {tabs.map((tab, index) => (
                        <TabsTrigger
                            key={tab.label + index}
                            value={tab.value}
                            variant="wallet"
                            className="gap-1.5 sm:gap-2 md:gap-3 text-xs sm:text-sm md:text-base lg:text-lg"
                        >
                            {tab.icon ? <LucideIcon name={tab.icon} className="w-4 h-4 sm:w-5 sm:h-5" /> : null}
                            {tab.label}
                        </TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>
        </div>
    );
};
