const CANOPY_SYMBOLS = [
    '/canopy-symbol-color.png',
    '/canopy-symbol-light.png',
    '/canopy-symbol-white.png',
] as const;

export function getCanopySymbol(index: number): string {
    return CANOPY_SYMBOLS[index % CANOPY_SYMBOLS.length];
}

export function getCanopySymbolByHash(input: string): string {
    let h = 0;
    for (let i = 0; i < input.length; i++) h = (h << 5) - h + input.charCodeAt(i);
    return CANOPY_SYMBOLS[Math.abs(h) % CANOPY_SYMBOLS.length];
}
