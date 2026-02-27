import React from 'react';

export const Footer = (): JSX.Element => {
    const links = [
        { label: 'Terms',     href: '#' },
        { label: 'Privacy',   href: '#' },
        { label: 'Security',  href: '#' },
        { label: 'Support',   href: '#' },
    ];

    return (
        <footer className="border-t border-border/40 mt-8">
            <div className="px-5 py-3">
                <div className="flex flex-wrap justify-center items-center gap-5">
                    {links.map(({ label, href }) => (
                        <a
                            key={label}
                            href={href}
                            className="text-muted-foreground hover:text-primary transition-colors duration-150 text-xs font-body whitespace-nowrap"
                        >
                            {label}
                        </a>
                    ))}
                    <span className="text-muted-foreground/40 text-xs font-mono ml-2">
                        v1.0
                    </span>
                </div>
            </div>
        </footer>
    );
};
