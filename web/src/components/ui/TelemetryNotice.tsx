import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@heroui/react";
import { Info } from "lucide-react";
import { Telemetry, type TelemetryNotice as TelemetryNoticeData, type TelemetryNoticeAction } from "@/lib/api";
import { TELEMETRY_STATEMENT_URL } from "@/lib/links";

export interface TelemetryNoticeProps {
  /** Navigates to /admin?section=telemetry. */
  onOpenSettings: () => void;
}

function destinationSuffix(kind: string): string {
  if (kind === "default") return ", the Gameplane project's telemetry service.";
  if (kind === "bundled") {
    return ", the receiver bundled with this install. Reports stay inside your cluster.";
  }
  return ", the destination your operator configured.";
}

const linkClass =
  "text-xs font-semibold text-accent-soft-foreground underline-offset-2 hover:underline";

/**
 * First-login telemetry notice (spec 022 FR-004, design frame U8Zugm). Mount
 * only for users holding config:manage. Renders while the notice is pending;
 * posts "seen" once on mount; every action acks, then hides the banner.
 */
export function TelemetryNotice({ onOpenSettings }: TelemetryNoticeProps) {
  const queryClient = useQueryClient();
  const [failed, setFailed] = useState(false);
  const seenPosted = useRef(false);
  const { data } = useQuery({
    queryKey: ["telemetry", "notice"],
    queryFn: () => Telemetry.notice(),
  });
  const pending = data?.pending === true;

  useEffect(() => {
    if (!pending || seenPosted.current) return;
    seenPosted.current = true;
    // Best effort: a failed "seen" only delays the schedule opening.
    void Telemetry.ack("seen").catch(() => {});
  }, [pending]);

  const ack = useMutation({
    mutationFn: (action: TelemetryNoticeAction) => Telemetry.ack(action),
    onMutate: () => setFailed(false),
    onSuccess: async () => {
      queryClient.setQueryData<TelemetryNoticeData>(["telemetry", "notice"], { pending: false });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["telemetry"] }),
        queryClient.invalidateQueries({ queryKey: ["admin-config"] }),
      ]);
    },
    onError: () => setFailed(true),
  });

  if (!pending) return null;
  const dest = data?.destination;

  return (
    <div className="px-4 pt-4 sm:px-6">
      <section
        aria-label="Telemetry notice"
        className="flex w-full gap-3 rounded-lg border border-accent-soft-foreground bg-accent-soft p-4"
      >
        <Info aria-hidden="true" className="h-5 w-5 shrink-0 text-accent-soft-foreground" />
        <div className="flex min-w-0 flex-1 flex-col gap-3">
          <p className="text-sm font-semibold text-accent-soft-foreground">
            Anonymous usage metrics are on for this install.
          </p>
          {dest?.host ? (
            <p className="text-[13px] text-foreground">
              Reports are sent to{" "}
              <span className="font-semibold text-accent-soft-foreground">{dest.host}</span>
              {destinationSuffix(dest.kind)}
            </p>
          ) : null}
          <div className="flex flex-col gap-4 sm:flex-row sm:gap-6">
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <p className="text-[13px] font-semibold text-foreground">Basic metrics</p>
              <p className="text-xs text-muted">
                Gameplane version, number of servers, number of templates.
              </p>
            </div>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <p className="text-[13px] font-semibold text-foreground">Extended metrics</p>
              <p className="text-xs text-muted">
                Kubernetes version, distribution, CPU architecture and node-count band; servers
                per official game module (others counted as custom); which optional features are
                in use; and a random install ID.
              </p>
            </div>
          </div>
          {failed && (
            <p role="alert" className="text-xs text-danger">
              Could not save your choice. Try again.
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="primary" size="sm" isDisabled={ack.isPending} onPress={() => ack.mutate("keep")}>
              Keep sharing
            </Button>
            <Button variant="outline" size="sm" isDisabled={ack.isPending} onPress={() => ack.mutate("extended-off")}>
              Turn off extended
            </Button>
            <Button variant="outline" size="sm" isDisabled={ack.isPending} onPress={() => ack.mutate("all-off")}>
              Turn off all
            </Button>
            <div className="ml-auto flex items-center gap-2.5">
              <button type="button" onClick={onOpenSettings} className={linkClass}>
                Open Telemetry settings
              </button>
              <span aria-hidden="true" className="select-none text-muted">·</span>
              <a href={TELEMETRY_STATEMENT_URL} target="_blank" rel="noopener noreferrer" className={linkClass}>
                Data-handling statement
              </a>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
