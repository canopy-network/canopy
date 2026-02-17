import React from 'react';

interface NetworkPeersProps {
    networkPeers: {
        totalPeers: number;
        connections: { in: number; out: number };
        peerId: string;
        networkAddress: string;
        publicKey: string;
        peers: Array<{
            address: { publicKey: string; netAddress: string };
            isOutbound: boolean;
            isValidator: boolean;
            isMustConnect: boolean;
            isTrusted: boolean;
            reputation: number;
        }>;
    };
}

export default function NetworkPeers({ networkPeers }: NetworkPeersProps): JSX.Element {
    return (
        <div
            className="rounded-xl border border-white/10 p-6"
            style={{ background: '#22232E' }}
        >
            <h2 className="text-white text-base font-semibold mb-4">Network Peers</h2>

            <div className="grid grid-cols-2 gap-4 mb-5">
                <div>
                    <div className="text-xs text-back mb-1">Total Peers</div>
                    <div className="text-2xl font-bold text-primary">{networkPeers.totalPeers}</div>
                </div>
                <div>
                    <div className="text-xs text-back mb-1">Connections</div>
                    <div className="flex items-center gap-2 text-sm font-medium">
                        <span className="text-primary">{networkPeers.connections.in} in</span>
                        <span className="text-back">/</span>
                        <span className="text-back">{networkPeers.connections.out} out</span>
                    </div>
                </div>
            </div>

            <div className="space-y-3">
                <div>
                    <div className="text-xs text-back mb-0.5">Peer ID</div>
                    <div className="text-sm font-mono text-white truncate">{networkPeers.peerId || '—'}</div>
                </div>
                <div>
                    <div className="text-xs text-back mb-0.5">Network Address</div>
                    <div className="text-sm font-mono text-white">{networkPeers.networkAddress || '—'}</div>
                </div>
            </div>
        </div>
    );
}
