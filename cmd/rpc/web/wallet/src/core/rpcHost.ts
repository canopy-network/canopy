export type RpcBase = "rpc" | "admin" | "root";

export function resolveRpcHost(chain: any, base: RpcBase = "rpc"): string {
  if (!chain?.rpc) return "";

  if (base === "admin") {
    return chain.rpc.admin ?? chain.rpc.base ?? chain.rpc.root ?? "";
  }

  if (base === "root") {
    return chain.rpc.root ?? chain.rpc.base ?? "";
  }

  return chain.rpc.base ?? chain.rpc.root ?? "";
}
