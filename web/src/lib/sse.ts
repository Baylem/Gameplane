// Thin reconnecting Server-Sent Events client for the API's /events
// stream. The API frames each Kubernetes watch event as
// {kind, eventType, object}; the dashboard uses this to invalidate
// TanStack Query caches (so views refresh without waiting for the next
// poll) and to feed the notifications panel.

export interface GameplaneEvent {
  // CRD path segment: "servers" | "templates" | "backups" | "schedules" | "restores"
  kind: string;
  // Kubernetes watch event type: "ADDED" | "MODIFIED" | "DELETED"
  eventType: string;
  object: { metadata?: { name?: string; namespace?: string } } & Record<string, unknown>;
}

export interface EventStreamOptions {
  onEvent: (ev: GameplaneEvent) => void;
  onError?: () => void;
  onReconnect?: () => void;
}

// openEventStream connects to /events and invokes onEvent for each parsed
// frame. Returns a disposer that closes the stream and stops reconnects.
// EventSource reconnects on transient drops on its own; we additionally
// re-open if the connection errors out and was closed.
// When the tab is hidden (background), the stream is closed to free the
// connection slot; it reconnects when the tab becomes visible again.
export function openEventStream(opts: EventStreamOptions): () => void {
  // No EventSource (e.g. jsdom/test, or an ancient browser) → no-op; the
  // dashboard's refetchInterval pollers keep data fresh as a fallback.
  if (typeof EventSource === "undefined") return () => {};

  let es: EventSource | null = null;
  let closed = false;
  // A tab opened in the background never gets a visibilitychange to tell
  // us it is hidden, so start from the current state and do not connect
  // until it becomes visible (the handler below then connects and calls
  // onReconnect).
  let hidden = typeof document !== "undefined" && document.hidden;
  let retry: ReturnType<typeof setTimeout> | undefined;

  function connect() {
    if (closed || hidden) return;
    es = new EventSource("/events", { withCredentials: true });
    es.onmessage = (e) => {
      try {
        opts.onEvent(JSON.parse(e.data) as GameplaneEvent);
      } catch {
        // Ignore malformed frames rather than tearing down the stream.
      }
    };
    es.onerror = () => {
      opts.onError?.();
      // EventSource auto-retries while open; if the browser closed it,
      // re-open after a short backoff.
      if (!closed && !hidden && es && es.readyState === EventSource.CLOSED) {
        es.close();
        retry = setTimeout(connect, 3000);
      }
    };
  }
  connect();

  function handleVisibilityChange() {
    if (document.hidden) {
      hidden = true;
      if (retry) clearTimeout(retry);
      es?.close();
      es = null;
    } else {
      hidden = false;
      if (!closed && !es) {
        connect();
        opts.onReconnect?.();
      }
    }
  }

  document.addEventListener("visibilitychange", handleVisibilityChange);

  return () => {
    closed = true;
    if (retry) clearTimeout(retry);
    es?.close();
    document.removeEventListener("visibilitychange", handleVisibilityChange);
  };
}

// queryKeyForKind maps an event's CRD kind to the TanStack Query key the
// dashboard caches it under, so a watch event invalidates the right view.
export function queryKeyForKind(kind: string): string[] | null {
  switch (kind) {
    case "servers":
      return ["servers"];
    case "templates":
      return ["templates"];
    case "backups":
      return ["backups"];
    case "schedules":
      return ["schedules"];
    case "restores":
      return ["restores"];
    default:
      return null;
  }
}
