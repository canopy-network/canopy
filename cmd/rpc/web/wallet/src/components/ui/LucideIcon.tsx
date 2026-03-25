import React, { Suspense } from 'react';
import dynamicIconImports from 'lucide-react/dynamicIconImports';
import { CircleHelp } from 'lucide-react';

type Props = { name?: string; className?: string };
type Importer = () => Promise<{ default: React.ComponentType<any> }>;
const LIB = dynamicIconImports as Record<string, Importer>;

const normalize = (n?: string) => {
    if (!n) return 'help-circle';
    return n
        .replace(/([a-z0-9])([A-Z])/g, '$1-$2') // separate uppercase letters with "-"
        .replace(/[_\s]+/g, '-') // convert spaces or underscores to "-"
        .toLowerCase()
        .trim();
};

const FALLBACKS = ['circle-help', 'help-circle', 'zap', 'circle', 'square'];
const FALLBACK_ICON: React.ComponentType<any> = CircleHelp;
const ICON_ALIASES: Record<string, string> = {
    'check-square': 'check',
    'square-check-big': 'check',
};

const cache = new Map<string, React.LazyExoticComponent<React.ComponentType<any>>>();

export function LucideIcon({ name = 'HelpCircle', className }: Props) {
    const normalizedName = normalize(name);
    const key = ICON_ALIASES[normalizedName] || normalizedName;

    const resolvedName =
        (LIB[key] && key) ||
        FALLBACKS.find(k => !!LIB[k]) ||
        Object.keys(LIB)[0];


    const importer = resolvedName ? LIB[resolvedName] : undefined;

    if (!importer || typeof importer !== 'function') {
        return <FALLBACK_ICON className={className} />;
    }

    let Icon = cache.get(resolvedName);
    if (!Icon) {
        const safeImporter: Importer = () =>
            importer().catch((error) => {
                if (typeof console !== 'undefined') {
                    console.warn(`[LucideIcon] Failed to load icon "${resolvedName}"`, error);
                }
                return Promise.resolve({ default: FALLBACK_ICON });
            });

        Icon = React.lazy(safeImporter);
        cache.set(resolvedName, Icon);
    }

    return (
        <Suspense fallback={<span className={className} />}>
            <Icon className={className} />
        </Suspense>
    );
}
