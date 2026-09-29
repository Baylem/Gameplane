import { useQuery } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";
import type { User } from "@/types";

// Total attempts for /users/me before the route guards give up and show a
// retry card. Three covers a rolling API restart without stalling the UI.
const ME_ATTEMPTS = 3;

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api<User>("/users/me"),
    // A 401 is a definitive answer: retrying only delays the /login redirect.
    // Anything else (API rollout, proxy hiccup, a stalled DB connection) is
    // transient, and must not be mistaken for "you are not allowed here".
    retry: (failureCount, err) =>
      !(err instanceof APIError && err.status === 401) && failureCount < ME_ATTEMPTS,
  });
}

/**
 * can reports whether the current user holds a permission. It mirrors the
 * server's rbac.Can: a cluster-wide ("*") grant of the permission (or the
 * "*" wildcard) suffices anywhere; for a namespaced action, a grant in the
 * target namespace also counts. When permissionsByCluster is present, cluster
 * boundaries are enforced (prevent cross-cluster privilege escalation). The
 * API is always the real enforcer — this only drives what the UI shows.
 *
 * When permissionsByCluster is absent (older API), falls back to the flat
 * permissions structure (cluster-agnostic).
 */
export function can(
  me: User | undefined,
  perm: string,
  ns?: string,
  cluster?: string,
): boolean {
  // Backward compatibility: when permissionsByCluster is absent, use flat structure.
  if (!me?.permissionsByCluster) {
    const perms = me?.permissions;
    if (!perms) return false;
    const has = (set: string[] | undefined) =>
      !!set && (set.includes("*") || set.includes(perm));
    if (has(perms["*"])) return true;
    if (ns && ns !== "*" && has(perms[ns])) return true;
    return false;
  }

  const perms = me.permissionsByCluster;

  // Helper: does user hold perm cluster-wide ("*" namespace) on cluster ck?
  const cwHolds = (ck: string): boolean => {
    const clusterPerms = perms[ck];
    if (!clusterPerms) return false;
    const nsPerms = clusterPerms["*"];
    if (!nsPerms) return false;
    return nsPerms.includes("*") || nsPerms.includes(perm);
  };

  if (!ns || ns === "*") {
    // Control-plane perm (no namespace): any cluster's cluster-wide binding
    // grants it.
    for (const ck of Object.keys(perms)) {
      if (cwHolds(ck)) {
        return true;
      }
    }
    return false;
  }

  // Namespaced perm: gated by target cluster (or "*" wildcard cluster).
  // Iterate through target cluster first, then wildcard, to match backend order.
  const targetClusters = cluster ? [cluster, "*"] : ["*"];
  for (const ck of targetClusters) {
    if (cwHolds(ck)) {
      return true;
    }
    const clusterPerms = perms[ck];
    if (!clusterPerms) continue;
    const nsPerms = clusterPerms[ns];
    if (!nsPerms) continue;
    if (nsPerms.includes("*") || nsPerms.includes(perm)) {
      return true;
    }
  }
  return false;
}
