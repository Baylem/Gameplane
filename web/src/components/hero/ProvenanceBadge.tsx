import React from "react";
import { Chip } from "@heroui/react";
import { Edit2, Package, Minus } from "lucide-react";

// ProvenanceBadge renders a data provenance indicator showing how a configuration value was sourced.
// Three variants: "overridden" (manual edit), "fromHelm" (Helm values), "notConfigured" (not set).
// Used in admin settings and configuration screens to track configuration origin.

export type ProvenanceType = "overridden" | "fromHelm" | "notConfigured";

export interface ProvenanceBadgeProps {
  type: ProvenanceType;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const provenanceConfig: Record<ProvenanceType, {
  icon: React.ReactNode;
  label: string;
  variant: "soft" | "tertiary";
  color: "default" | "danger" | "warning" | "success";
}> = {
  // Neutral/outlined per R65Xyx (no fill, bordered, muted text/icon) —
  // this design was moved off the pink chip--soft pill; see the scoped
  // [data-type="overridden"] rule in globals.css for the border/muted color.
  overridden: {
    icon: <Edit2 className="h-3 w-3" />,
    label: "Overridden in dashboard",
    variant: "tertiary",
    color: "default",
  },
  fromHelm: {
    icon: <Package className="h-3 w-3" />,
    label: "From Helm values",
    variant: "soft",
    color: "default",
  },
  notConfigured: {
    icon: <Minus className="h-3 w-3" />,
    label: "Not configured",
    variant: "soft",
    color: "danger",
  },
};

export function ProvenanceBadge({
  type,
  size = "sm",
  className,
}: ProvenanceBadgeProps) {
  const config = provenanceConfig[type];

  return (
    <Chip
      variant={config.variant}
      color={config.color}
      size={size}
      data-type={type}
      className={className}
    >
      <div className="flex items-center gap-1">
        {config.icon}
        <span className="text-xs font-medium">{config.label}</span>
      </div>
    </Chip>
  );
}
