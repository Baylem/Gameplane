import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { RouterProvider as AriaRouterProvider } from "react-aria-components";

import { routeTree } from "@/router/tree";
import { createAriaNavigate, createAriaUseHref } from "@/lib/ariaRouter";
import "@/styles/globals.css";

// defaultPendingMs/defaultPendingMinMs (TanStack Router's own defaults)
// gate the root route's pendingComponent (AppLoadingSkeleton, design
// N13Xud) — a 500ms show-delay avoids a flash on fast loads, and a 500ms
// minimum-visible avoids it flickering back off immediately after.
const router = createRouter({
  routeTree,
  defaultPendingMs: 500,
  defaultPendingMinMs: 500,
});

declare module "@tanstack/react-router" {
  interface Register { router: typeof router }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, refetchOnWindowFocus: false },
  },
});

async function bootstrap(): Promise<void> {
  // Dynamic-import the MSW worker only when the E2E mock flag is set.
  // VITE_E2E_MOCK comes from web/.env.mock, loaded by `vite --mode mock`
  // (Playwright's webServer in mock target). The branch is statically
  // dead in production (vite resolves import.meta.env at build time),
  // so the bundle never ships MSW.
  if (import.meta.env.VITE_E2E_MOCK === "true") {
    const { startMSW } = await import("@/test/browser-msw");
    await startMSW();
  }

  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <AriaRouterProvider
          navigate={createAriaNavigate(router)}
          useHref={createAriaUseHref(router)}
        >
          <RouterProvider router={router} />
        </AriaRouterProvider>
      </QueryClientProvider>
    </React.StrictMode>,
  );
}

void bootstrap();
