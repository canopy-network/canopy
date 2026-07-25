export type RpcBase = "rpc" | "admin";

export function resolveRpcHost(chain: any, base: RpcBase = "rpc"): string {
  if (!chain?.rpc) return "";

  if (base === "admin") {
    return chain.rpc.admin ?? chain.rpc.base ?? "";
  }

  return chain.rpc.base ?? "";
}
