import {isAddress, getAddress} from 'viem'

export function normalizeEvmAddress(input: string) {
    if (!input) return {ok: false as const, value: '', reason: 'empty'};
    const s = input.startsWith('0x') ? input : `0x${input}`;
    const ok = isAddress(s, {strict: false});
    return ok ? {ok: true as const, value: getAddress(s)} : {ok: false as const, value: '', reason: 'invalid-evm'}
}
