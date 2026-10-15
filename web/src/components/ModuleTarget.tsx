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
import { useOptionalIntl } from "@/lib/i18n/format";

const unauthorized = (error: unknown) => error instanceof APIError && error.status === 401;

// Do not mount workload queries until the installation and target are known.
// Combined installations retain their existing local module catalog.
export function ModuleTarget({ children, loadingMessage: loadingMessageProp }: { children: (cluster: string, canManage: boolean, clusterDisplayName: string) => ReactNode; loadingMessage?: string }) {
  const intl = useOptionalIntl();
  const loadingMessage = loadingMessageProp ?? intl.formatMessage({ id: "web.components.module_target.loading_installation", defaultMessage: "Loading installation…", description: "Module target: loading message while the installation settings load, used when no loadingMessage is passed" });
  const installation = useInstallation();
  const identity = useMe();
  const selected = useCurrentCluster();
  const registry = useQuery({
    queryKey: ["clusters"], queryFn: () => Clusters.list(),
    enabled: installation.data?.standalone === true,
  });
  if (installation.isPending) return <LoadingCard message={loadingMessage} />;
  if (installation.isError && (!installation.data || unauthorized(installation.error))) return <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.couldn_t_load_installation_capabilities", defaultMessage: "Couldn't load installation capabilities.", description: "Module target: message of ErrorCard" })} onRetry={() => void installation.refetch()} />;
  const renderTarget = (cluster: string, clusterDisplayName = cluster) => {
    if (identity.isPending) return <LoadingCard message={intl.formatMessage({ id: "web.components.module_target.loading_permissions", defaultMessage: "Loading permissions\u2026", description: "Module target: loading message while permissions load" })} />;
    if (identity.isError && (!identity.data || unauthorized(identity.error))) return <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.permissions_error", defaultMessage: "Couldn't load your permissions.", description: "Module target: error message when permissions fail to load" })} onRetry={() => void identity.refetch()} />;
    return <>
      {identity.isError && <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.permissions_error", defaultMessage: "Couldn't load your permissions.", description: "Module target: error message when permissions fail to load" })} onRetry={() => void identity.refetch()} />}
      {can(identity.data, "modules:read", "*", cluster)
        ? children(cluster, can(identity.data, "modules:manage", "*", cluster), clusterDisplayName)
        : <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.no_permission", defaultMessage: "You don't have permission to view modules on this cluster.", description: "Module target: message shown when the user cannot view modules on this cluster" })} />}
    </>;
  };
  const installationError = installation.isError && <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.couldn_t_load_installation_capabilities", defaultMessage: "Couldn't load installation capabilities.", description: "Module target: message of ErrorCard" })} onRetry={() => void installation.refetch()} />;
  if (!installation.data.standalone) return <>{installationError}{renderTarget("local")}</>;
  const target = registry.data?.items.find((cluster) => cluster.name === selected);
  return <div className="space-y-4">
    <ClusterSelector />
    {installationError}
    {registry.isError && registry.data && !unauthorized(registry.error) && <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.couldn_t_load_registered_clusters", defaultMessage: "Couldn't load registered clusters.", description: "Module target: message of ErrorCard" })} onRetry={() => void registry.refetch()} />}
    {registry.isPending ? <LoadingCard message={intl.formatMessage({ id: "web.components.module_target.loading_clusters", defaultMessage: "Loading clusters…", description: "Module target: message of LoadingCard" })} />
      : registry.isError && (!registry.data || unauthorized(registry.error)) ? <ErrorCard message={intl.formatMessage({ id: "web.components.module_target.couldn_t_load_registered_clusters", defaultMessage: "Couldn't load registered clusters.", description: "Module target: message of ErrorCard" })} onRetry={() => void registry.refetch()} />
      : target ? renderTarget(target.name, target.displayName || target.name)
      : <Card><CardContent className="space-y-2 p-6">
        <h2 className="font-medium">{intl.formatMessage({ id: "web.components.module_target.select_a_workload_cluster", defaultMessage: "Select a workload cluster", description: "Module target: text in h2" })}</h2>
        <p className="text-sm text-muted">{intl.formatMessage({ id: "web.components.module_target.modules_and_their_sources_are_managed", defaultMessage: "Modules and their sources are managed by the operator on a registered workload cluster.", description: "Module target: text in p" })}</p>
        <Link to="/clusters" className="text-sm text-primary">{intl.formatMessage({ id: "web.components.module_target.manage_clusters", defaultMessage: "Manage clusters", description: "Module target: text in Link" })}</Link>
      </CardContent></Card>}
  </div>;
}
