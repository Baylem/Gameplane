import { describe, expect, it } from "vitest";
import type { GameServer } from "@/types";
import { mergeDraftOntoLatest, SettingsConflictError } from "./settingsMerge";

function server(): GameServer {
  return {
    metadata: { name: "survival", resourceVersion: "100", labels: { team: "old", remove: "yes" } },
    spec: { templateRef: { name: "minecraft-java" }, suspend: false, version: "1.20", config: { MAX_PLAYERS: "16" } },
  };
}

describe("mergeDraftOntoLatest", () => {
  it("preserves a concurrent stop, version change, nested config edit and label addition", () => {
    const baseline = server();
    const draft = structuredClone(baseline);
    draft.spec.config!.MAX_PLAYERS = "20";
    draft.metadata.labels!.team = "new";
    const latest = structuredClone(baseline);
    latest.metadata.resourceVersion = "200";
    latest.metadata.labels!.operator = "managed";
    latest.spec.suspend = true;
    latest.spec.version = "1.21";
    latest.spec.config!.MOTD = "Hello";
    Object.assign(latest.spec, { futureOption: { enabled: true } });

    const merged = mergeDraftOntoLatest(draft, baseline, latest);
    expect(merged.spec).toEqual({ templateRef: { name: "minecraft-java" }, suspend: true, version: "1.21", config: { MAX_PLAYERS: "20", MOTD: "Hello" }, futureOption: { enabled: true } });
    expect(merged.metadata.labels).toEqual({ team: "new", remove: "yes", operator: "managed" });
    expect(merged.metadata.resourceVersion).toBe("200");
    expect(latest.spec.config!.MAX_PLAYERS).toBe("16");
  });

  it("rejects divergent changes to the same spec field or metadata key", () => {
    for (const field of ["config", "label", "annotation"] as const) {
      const baseline = server();
      baseline.metadata.annotations = { description: "old" };
      const draft = structuredClone(baseline);
      const latest = structuredClone(baseline);
      if (field === "config") {
        draft.spec.config!.MAX_PLAYERS = "20";
        latest.spec.config!.MAX_PLAYERS = "24";
      } else if (field === "label") {
        draft.metadata.labels!.team = "new";
        latest.metadata.labels!.team = "other";
      } else {
        draft.metadata.annotations!.description = "new";
        latest.metadata.annotations!.description = "other";
      }
      expect(() => mergeDraftOntoLatest(draft, baseline, latest)).toThrow(SettingsConflictError);
    }
  });

  it("removes baseline map keys without removing concurrently added keys", () => {
    const baseline = server();
    baseline.metadata.annotations = { description: "old" };
    const draft = structuredClone(baseline);
    delete draft.spec.config;
    delete draft.metadata.labels;
    delete draft.metadata.annotations;
    const latest = structuredClone(baseline);
    latest.spec.config!.MOTD = "Hello";
    latest.metadata.labels!.operator = "managed";
    latest.metadata.annotations!.operator = "managed";

    const merged = mergeDraftOntoLatest(draft, baseline, latest);
    expect(merged.spec.config).toEqual({ MOTD: "Hello" });
    expect(merged.metadata.labels).toEqual({ operator: "managed" });
    expect(merged.metadata.annotations).toEqual({ operator: "managed" });
  });

  it("deletes cleared optional fields and accepts identical concurrent edits", () => {
    const baseline = server();
    const draft = structuredClone(baseline);
    delete draft.spec.version;
    delete draft.spec.config;
    draft.spec.suspend = true;
    const latest = structuredClone(baseline);
    latest.spec.suspend = true;
    const merged = mergeDraftOntoLatest(draft, baseline, latest);
    expect(merged.spec.version).toBeUndefined();
    expect(merged.spec.config).toBeUndefined();
    expect(merged.spec.suspend).toBe(true);
  });

  it("conflicts when a deleted key changed concurrently or both writers replaced an array", () => {
    const baseline = server();
    baseline.spec.env = [{ name: "MODE", value: "old" }];
    const draft = structuredClone(baseline);
    delete draft.spec.config;
    const latest = structuredClone(baseline);
    latest.spec.config!.MAX_PLAYERS = "24";
    expect(() => mergeDraftOntoLatest(draft, baseline, latest)).toThrow(SettingsConflictError);
    draft.spec.config = baseline.spec.config;
    latest.spec.config = baseline.spec.config;
    draft.spec.env = [{ name: "MODE", value: "mine" }];
    latest.spec.env = [{ name: "MODE", value: "theirs" }];
    expect(() => mergeDraftOntoLatest(draft, baseline, latest)).toThrow(SettingsConflictError);
  });

  it("combines independent keys when both writers create a previously absent map", () => {
    const baseline = server();
    delete baseline.spec.config;
    const draft = structuredClone(baseline);
    draft.spec.config = { MAX_PLAYERS: "20" };
    const latest = structuredClone(baseline);
    latest.spec.config = { MOTD: "Hello" };
    expect(mergeDraftOntoLatest(draft, baseline, latest).spec.config).toEqual({ MAX_PLAYERS: "20", MOTD: "Hello" });
  });

  it("deletes a label named constructor without treating inherited properties as values", () => {
    const baseline = server();
    Object.assign(baseline.metadata.labels!, { constructor: "owned" });
    const draft = structuredClone(baseline);
    Reflect.deleteProperty(draft.metadata.labels!, "constructor");
    const latest = structuredClone(baseline);
    latest.metadata.labels!.operator = "managed";
    expect(mergeDraftOntoLatest(draft, baseline, latest).metadata.labels).toEqual({ team: "old", remove: "yes", operator: "managed" });
  });
});
