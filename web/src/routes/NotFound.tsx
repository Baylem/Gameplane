import { Compass } from "lucide-react";
import { Link } from "@tanstack/react-router";

// Rendered inside the app shell for any URL no route claims. The auth guard
// lives in AppLayout (401 -> /login), so this page carries no cluster data.
export function NotFoundPage() {
  return (
    <div className="flex h-full min-h-[60vh] items-center justify-center p-6">
      <div className="flex max-w-md flex-col items-center text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted/15 text-muted">
          <Compass className="h-7 w-7" aria-hidden="true" />
        </div>
        <h1 className="pt-4 text-xl font-semibold text-fg">Page not found</h1>
        <p className="pt-2 text-sm text-muted">
          The page you&apos;re looking for doesn&apos;t exist or has moved.
        </p>
        <Link
          to="/"
          className="mt-5 inline-block rounded-full bg-primary px-5 py-2.5 text-sm font-medium text-primary-fg hover:opacity-90"
        >
          Go to dashboard
        </Link>
      </div>
    </div>
  );
}
