import React, { Suspense } from 'react';
import dynamicIconImports from 'lucide-react/dynamicIconImports';

type Props = { name?: string; className?: string };
type Importer = () => Promise<{ default: React.ComponentType<any> }>;
const LIB = dynamicIconImports as Record<string, Importer>;

const normalize = (n?: string) =>
    (!n ? 'HelpCircle' : n)
        .replace(/[-_ ]+/g, ' ')
        .toLowerCase()
        .replace(/\s+/g, '').trim(); // "qr-code" -> "QrCode", "send" -> "Send"

const FALLBACKS = ['HelpCircle', 'Zap', 'Circle', 'Square']; // keys que existen en casi todas las versiones

const cache = new Map<string, React.LazyExoticComponent<React.ComponentType<any>>>();

export function LucideIcon({ name = 'HelpCircle', className }: Props) {
    const key = normalize(name);

    const resolvedName =
        (LIB[key] && key) ||
        FALLBACKS.find(k => !!LIB[k]) ||
        Object.keys(LIB)[0];


    const importer = resolvedName ? LIB[resolvedName] : undefined;

    if (!importer || typeof importer !== 'function') {
        return <span className={className} />;
    }

    let Icon = cache.get(resolvedName);
    if (!Icon) {
        Icon = React.lazy(importer);
        cache.set(resolvedName, Icon);
    }

    return (
        <Suspense fallback={<span className={className} />}>
            <Icon className={className} />
        </Suspense>
    );
}
