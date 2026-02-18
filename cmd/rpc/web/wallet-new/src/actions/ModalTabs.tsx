import React from 'react';
import {LucideIcon} from "@/components/ui/LucideIcon";

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
    return (
        <div className="flex items-center justify-between mb-3 sm:mb-4 md:mb-6 border-b border-border/50 px-1 sm:px-2 md:px-3">
            <div className="flex items-center gap-2 sm:gap-4 md:gap-6 overflow-x-auto scrollbar-hide w-full" key={tabs.length}>
                {tabs.map((tab, index) => (
                    <button
                        key={tab.label + index}
                        onClick={() => onTabChange?.(tab)}
                        className={`flex gap-1.5 sm:gap-2 md:gap-3 items-center px-2 sm:px-3 md:px-4 py-2 text-xs sm:text-sm md:text-base lg:text-lg font-medium md:font-normal whitespace-nowrap flex-shrink-0 ${activeTab?.value === tab.value ? 'border-b-2 border-canopy-50 text-primary' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        <LucideIcon name={tab.icon} className="w-4 h-4 sm:w-5 sm:h-5"/>
                        {tab.label}
                    </button>
                ))}
            </div>
        </div>
    );
};
