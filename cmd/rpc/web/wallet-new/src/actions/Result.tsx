import React from 'react';

function ResultInner({ message, link, onDone }:{ message: string; link?: { label: string; href: string }; onDone: () => void }) {
    return (
        <div className="space-y-4">
            <div className="bg-neutral-900 border border-neutral-800 rounded p-4">
                <p>{message}</p>
                {link && <p className="mt-2"><a className="text-emerald-400 underline" href={link.href}>{link.label}</a></p>}
            </div>
            <button onClick={onDone} className="px-3 py-2 bg-neutral-800 rounded">Done</button>
        </div>
    );
}
export default React.memo(ResultInner);
