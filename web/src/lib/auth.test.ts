import { describe, it, expect } from "vitest";
import { can } from "./auth";
import type { User } from "@/types";

const u = (role: User["role"]): User => ({
  id: 1,
  username: "x",
  displayName: "X",
  email: "x@x",
  role,
});

const withPerms = (perms: Record<string, string[]>): User => ({
  ...u("custom"),
  permissions: perms,
});

const withPermsByCluster = (
  perms: Record<string, Record<string, string[]>>,
): User => ({
  ...u("custom"),
  permissionsByCluster: perms,
});

describe("can", () => {
  it("is false for an undefined user or no permissions", () => {
    expect(can(undefined, "servers:read")).toBe(false);
    expect(can(u("viewer"), "servers:read")).toBe(false);
  });
  it("grants any permission to a cluster-wide wildcard", () => {
    const admin = withPerms({ "*": ["*"] });
    expect(can(admin, "users:manage")).toBe(true);
    expect(can(admin, "anything:at:all")).toBe(true);
  });
  it("grants a specific cluster-wide permission", () => {
    const op = withPerms({ "*": ["servers:write", "servers:read"] });
    expect(can(op, "servers:write")).toBe(true);
    expect(can(op, "users:manage")).toBe(false);
  });
  it("honors a namespace-scoped grant only in that namespace", () => {
    const nsOp = withPerms({ "team-a": ["servers:write"] });
    expect(can(nsOp, "servers:write", "team-a")).toBe(true);
    expect(can(nsOp, "servers:write", "team-b")).toBe(false);
    // No cluster-wide grant: the bare check (no ns) is false.
    expect(can(nsOp, "servers:write")).toBe(false);
  });

  // Tests for cluster-aware permissionsByCluster structure (new behavior)
  it("uses permissionsByCluster when available", () => {
    const clusterAdmin = withPermsByCluster({
      local: { "*": ["*"] },
    });
    expect(can(clusterAdmin, "users:manage")).toBe(true);
    expect(can(clusterAdmin, "servers:write")).toBe(true);
  });

  it("denies namespace-scoped perms when granted on different cluster", () => {
    // Grant servers:write only on cluster "b"
    const user = withPermsByCluster({
      b: { "*": ["servers:write"] },
    });
    // Should be denied on cluster "a"
    expect(can(user, "servers:write", "game-ns", "a")).toBe(false);
    // Should be allowed on cluster "b"
    expect(can(user, "servers:write", "game-ns", "b")).toBe(true);
  });

  it("allows wildcard cluster to grant perms on any cluster", () => {
    // Grant servers:write on wildcard "*" cluster
    const user = withPermsByCluster({
      "*": { "*": ["servers:write"] },
    });
    // Should be allowed on any target cluster
    expect(can(user, "servers:write", "game-ns", "a")).toBe(true);
    expect(can(user, "servers:write", "game-ns", "b")).toBe(true);
  });

  it("honors namespace-specific grants within a cluster", () => {
    const user = withPermsByCluster({
      local: {
        "team-a": ["servers:write"],
        "team-b": ["servers:read"],
      },
    });
    expect(can(user, "servers:write", "team-a", "local")).toBe(true);
    expect(can(user, "servers:write", "team-b", "local")).toBe(false);
    expect(can(user, "servers:read", "team-b", "local")).toBe(true);
  });

  it("allows wildcard namespace to grant all perms in that cluster", () => {
    const user = withPermsByCluster({
      local: {
        "*": ["servers:write"],
      },
    });
    // Cluster-wide grant should match any namespace
    expect(can(user, "servers:write", "any-namespace", "local")).toBe(true);
  });

  it("allows wildcard permission to match any action", () => {
    const admin = withPermsByCluster({
      local: {
        "team-a": ["*"],
      },
    });
    expect(can(admin, "servers:write", "team-a", "local")).toBe(true);
    expect(can(admin, "captures:manage", "team-a", "local")).toBe(true);
  });

  it("falls back to flat permissions when permissionsByCluster is absent", () => {
    // Old-style permission structure without cluster info
    const user = withPerms({
      "*": ["servers:write"],
      "team-a": ["captures:manage"],
    });
    // Should still work with old behavior
    expect(can(user, "servers:write")).toBe(true);
    expect(can(user, "captures:manage", "team-a")).toBe(true);
  });

  it("denies a control-plane perm that no cluster grants cluster-wide", () => {
    expect(
      can(
        withPermsByCluster({ b: { "team-a": ["users:manage"] } }),
        "users:manage"
      )
    ).toBe(false);
  });

  it("grants a control-plane perm from any cluster's cluster-wide binding", () => {
    expect(
      can(
        withPermsByCluster({ b: { "*": ["users:manage"] } }),
        "users:manage"
      )
    ).toBe(true);
  });

  it("does not let a wildcard or missing namespace leak across clusters", () => {
    const user = withPermsByCluster({ b: { "*": ["servers:write"] } });
    expect(can(user, "servers:write", undefined, "a")).toBe(false);
    expect(can(user, "servers:write", "*", "a")).toBe(false);
    expect(can(user, "servers:write", "*", "b")).toBe(true);
  });

  it("matches a namespace grant only on its own cluster", () => {
    const user = withPermsByCluster({
      b: { "team-a": ["servers:write"] },
    });
    expect(can(user, "servers:write", "team-a", "a")).toBe(false);
    expect(can(user, "servers:write", "team-a", "b")).toBe(true);
    expect(can(user, "servers:write", "*", "b")).toBe(false);
  });

  it("checks only the wildcard cluster for a namespace without a cluster", () => {
    expect(
      can(
        withPermsByCluster({ b: { "team-a": ["servers:write"] } }),
        "servers:write",
        "team-a"
      )
    ).toBe(false);
    expect(
      can(
        withPermsByCluster({ "*": { "team-a": ["servers:write"] } }),
        "servers:write",
        "team-a"
      )
    ).toBe(true);
  });
});
