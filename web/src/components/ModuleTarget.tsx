import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card, CardContent } from "@heroui/react";
import { Clusters } from "@/lib/endpoints";
import { useCurrentCluster } from "@/lib/cluster";
import { useInstallation } from "@/lib/useInstallation";
import { ClusterSelector } from "@/components/ClusterSelector";
import { ErrorCard } from "@/components/ui/ErrorCard";
import { LoadingCard } from "@/components/ui/LoadingCard";
import { useMe, can } from "@/lib/auth";
import { APIError } from "@/lib/api";

const unauthorized = (error: unknown) => error instanceof APIError && error.status === 401;

// Do not mount workload queries until the installation and target are known.
// Combined installations retain their existing local module catalog.
export function ModuleTarget({ children, loadingMessage = "Loading installation…" }: { children: (cluster: string, canManage: boolean, clusterDisplayName: string) => ReactNode; loadingMessage?: string }) {
  const installation = useInstallation();
  const identity = useMe();
  const selected = useCurrentCluster();
  const registry = useQuery({
    queryKey: ["clusters"], queryFn: () => Clusters.list(),
    enabled: installation.data?.standalone === true,
  });
  if (installation.isPending) return <LoadingCard message={loadingMessage} />;
  if (installation.isError && (!installation.data || unauthorized(installation.error))) return <ErrorCard message="Couldn't load installation capabilities." onRetry={() => void installation.refetch()} />;
  const renderTarget = (cluster: string, clusterDisplayName = cluster) => {
    if (identity.isPending) return <LoadingCard message="Loading permissions…" />;
    if (identity.isError && (!identity.data || unauthorized(identity.error))) return <ErrorCard message="Couldn't load your permissions." onRetry={() => void identity.refetch()} />;
    return <>
      {identity.isError && <ErrorCard message="Couldn't load your permissions." onRetry={() => void identity.refetch()} />}
      {can(identity.data, "modules:read", "*", cluster)
        ? children(cluster, can(identity.data, "modules:manage", "*", cluster), clusterDisplayName)
        : <ErrorCard message="You don't have permission to view modules on this cluster." />}
    </>;
  };
  const installationError = installation.isError && <ErrorCard message="Couldn't load installation capabilities." onRetry={() => void installation.refetch()} />;
  if (!installation.data.standalone) return <>{installationError}{renderTarget("local")}</>;
  const target = registry.data?.items.find((cluster) => cluster.name === selected);
  return <div className="space-y-4">
    <ClusterSelector />
    {installationError}
    {registry.isError && registry.data && !unauthorized(registry.error) && <ErrorCard message="Couldn't load registered clusters." onRetry={() => void registry.refetch()} />}
    {registry.isPending ? <LoadingCard message="Loading clusters…" />
      : registry.isError && (!registry.data || unauthorized(registry.error)) ? <ErrorCard message="Couldn't load registered clusters." onRetry={() => void registry.refetch()} />
      : target ? renderTarget(target.name, target.displayName || target.name)
      : <Card><CardContent className="space-y-2 p-6">
        <h2 className="font-medium">Select a workload cluster</h2>
        <p className="text-sm text-muted">Modules and their sources are managed by the operator on a registered workload cluster.</p>
        <Link to="/clusters" className="text-sm text-primary">Manage clusters</Link>
      </CardContent></Card>}
  </div>;
}
