import { canopyIconSvg, getCanopyAccent } from "@/lib/utils/brand";

function encodeCanopyIcon(color: string): string {
    return `data:image/svg+xml;utf8,${encodeURIComponent(canopyIconSvg(color))}`;
}

export function getCanopySymbol(index: number): string {
    return encodeCanopyIcon(getCanopyAccent(index));
}

export function getCanopySymbolByHash(input: string): string {
    return encodeCanopyIcon(getCanopyAccent(input));
}
