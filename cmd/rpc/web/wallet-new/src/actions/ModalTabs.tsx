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
        <div className="flex items-center justify-between mb-6 border-b border-bg-accent/50 px-3">
            <div className="flex items-center gap-6" key={tabs.length}>
                {tabs.map((tab, index) => (

                    <button
                        key={tab.label + index}
                        onClick={() => onTabChange?.(tab)}
                        className={`flex gap-3  items-center px-4 py-2 text-sm lg:text-lg font-medium md:font-normal ${activeTab?.value === tab.value ? 'border-b border-canopy-50 text-primary' : 'text-text-muted'}`}
                    >
                        <LucideIcon name={tab.icon}/>

                        {tab.label}
                    </button>

                ))}
            </div>
        </div>
    );
};
