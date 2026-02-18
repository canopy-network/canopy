import React from 'react';

export const Footer = (): JSX.Element => {
    const links = [
        { label: 'Terms of Service', href: '#' },
        { label: 'Privacy Policy',   href: '#' },
        { label: 'Security Guide',   href: '#' },
        { label: 'Support',          href: '#' },
    ];

    return (
        <footer className="border-t border-border/60 mt-8 w-full">
            <div className="px-4 py-4 sm:px-6">
                <div className="flex flex-wrap justify-center items-center gap-4 sm:gap-8">
                    {links.map(({ label, href }) => (
                        <a
                            key={label}
                            href={href}
                            className="text-muted-foreground hover:text-primary transition-colors duration-150 text-xs sm:text-sm font-medium whitespace-nowrap"
                        >
                            {label}
                        </a>
                    ))}
                </div>
            </div>
        </footer>
    );
};
