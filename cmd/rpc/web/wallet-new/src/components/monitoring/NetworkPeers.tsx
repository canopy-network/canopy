import { useManifest } from '@/hooks/useManifest';
import React from 'react';

interface NetworkPeersProps {
    networkPeers: {
        totalPeers: number;
        connections: { in: number; out: number };
        peerId: string;
        networkAddress: string;
        publicKey: string;
        peers: Array<{
            address: {
                publicKey: string;
                netAddress: string;
            };
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
        <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
            <h2 className="text-text-primary text-lg font-bold mb-4">Network Peers</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                    <div className="text-text-muted text-xs">Total Peers</div>
                    <div className="text-text-accent text-2xl font-bold">{networkPeers.totalPeers}</div>
                </div>
                <div>
                    <div className="text-text-muted text-xs">Connections</div>
                    <div className="text-text-primary text-sm">
                        <span className="text-primary">{networkPeers.connections.in} in</span>  / <span className="text-blue-500">{networkPeers.connections.out} Out</span>
                    </div>
                </div>
            </div>
            <div className="space-y-2">
                <div className="md:w-4/12 w-full">
                    <div className="text-text-muted text-xs">Peer ID</div>
                    <div className="text-text-primary font-mono text-sm truncate">{networkPeers.peerId}</div>
                </div>
                <div>
                    <div className="text-text-muted text-xs">Network Address</div>
                    <div className="text-text-primary font-mono text-sm">{networkPeers.networkAddress}</div>
                </div>
            </div>
        </div>
    );
}
