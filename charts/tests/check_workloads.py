"""CI-only resource override, management placement and disruption checks."""
import json
from pathlib import Path
import shutil
import tempfile

import yaml

from check_gateway import CHART, one, render, settings


def main():
    defaults = render()
    assert not any(obj["kind"] == "PodDisruptionBudget" for obj in defaults)
    args = one(defaults, "Deployment", "gameplane-operator")["spec"]["template"]["spec"]["containers"][0]["args"]
    assert not any(arg.startswith("--tunnel-resources=") for arg in args)
    placement = {
        "nodeSelector": {"gameplane.local/pool": "management"},
        "tolerations": [{"key": "gameplane.local/pool", "operator": "Equal", "value": "management", "effect": "NoSchedule"}],
        "affinity": {"podAntiAffinity": {"preferredDuringSchedulingIgnoredDuringExecution": [{"weight": 100, "podAffinityTerm": {"topologyKey": "kubernetes.io/hostname", "labelSelector": {"matchLabels": {"app.kubernetes.io/instance": "gameplane"}}}}]}},
        "topologySpreadConstraints": [{"maxSkew": 1, "topologyKey": "kubernetes.io/hostname", "whenUnsatisfiable": "DoNotSchedule", "labelSelector": {"matchLabels": {"app.kubernetes.io/instance": "gameplane"}}}],
        "priorityClassName": "management",
    }
    component = dict(placement, pdb={"enabled": True, "maxUnavailable": 1})
    gateway = dict(settings(), **component)
    values = {
        "operator": dict(component, replicas=2, tunnelResources={"requests": {}, "limits": {}}, agentResources={"requests": {"cpu": "80m"}}),
        "api": dict(component, audit={"webhook": {"syslogBridge": dict(component, enabled=True, syslog={"addr": "logs:514"})}}, telemetry={"receiver": dict(component, enabled=True)}),
        "web": component, "mcpServer": dict(component, enabled=True), "gateway": gateway,
        "crds": {"autoApply": placement},
    }
    production = render(values, upgrade=True)
    for name in ["operator", "api", "web", "mcp-server", "gateway", "audit-syslog-bridge", "telemetry-receiver"]:
        deployment = one(production, "Deployment", "gameplane-" + name)
        spec = deployment["spec"]["template"]["spec"]
        for key, expected in placement.items():
            assert spec[key] == expected, (name, key)
        pdb = one(production, "PodDisruptionBudget", "gameplane-" + name)["spec"]
        assert pdb["maxUnavailable"] == 1 and "minAvailable" not in pdb
        assert pdb["selector"]["matchLabels"].items() <= deployment["spec"]["template"]["metadata"]["labels"].items()
    hook = one(production, "Job", "gameplane-crd-apply")["spec"]["template"]["spec"]
    for key, expected in placement.items():
        assert hook[key] == expected
    args = one(production, "Deployment", "gameplane-operator")["spec"]["template"]["spec"]["containers"][0]["args"]
    overrides = {arg.split("=", 1)[0]: json.loads(arg.split("=", 1)[1]) for arg in args if arg.startswith(("--agent-resources=", "--tunnel-resources="))}
    assert overrides["--tunnel-resources"] == {"requests": {}, "limits": {}}
    assert overrides["--agent-resources"] == {"requests": {"cpu": "80m"}}
    render({"api": {"pdb": {"enabled": True, "minAvailable": 1}}}, failure="api.pdb must allow the singleton API to be evicted")
    render({"api": {"pdb": {"enabled": True, "maxUnavailable": 0}}}, failure="api.pdb must allow the singleton API to be evicted")
    render({"operator": {"replicas": 2, "leaderElect": False}}, failure="operator.leaderElect must be enabled")
    single = render({"operator": {"leaderElect": False}})
    assert one(single, "Deployment", "gameplane-operator")["spec"]["strategy"] == {"type": "Recreate", "rollingUpdate": None}
    removed = render({"operator": {"agentResources": None, "tunnelResources": {}}})
    args = one(removed, "Deployment", "gameplane-operator")["spec"]["template"]["spec"]["containers"][0]["args"]
    assert not any(arg.startswith("--agent-resources=") for arg in args)
    assert "--tunnel-resources={}" in args
    disabled = render({"api": {"enabled": False, "pdb": {"enabled": True}}, "web": {"pdb": {"enabled": True}}})
    assert not any(obj["kind"] == "PodDisruptionBudget" for obj in disabled)
    profile = yaml.safe_load((CHART / "values-production.example.yaml").read_text())
    example = render(profile, upgrade=True)
    assert one(example, "Deployment", "gameplane-operator")["spec"]["replicas"] == 2
    assert one(example, "PodDisruptionBudget", "gameplane-api")["spec"]["maxUnavailable"] == 1
    # Helm --reuse-values may not contain new keys. Omission must retain binary
    # resource defaults and empty placement without indexing a missing map.
    with tempfile.TemporaryDirectory() as directory:
        old_chart = Path(directory) / "chart"
        shutil.copytree(CHART, old_chart)
        old_values = yaml.safe_load((old_chart / "values.yaml").read_text())
        for key in ["agentResources", "sentinelResources", "tunnelResources", "backupResources", "pdb", *placement]:
            old_values["operator"].pop(key, None)
        (old_chart / "values.yaml").write_text(yaml.safe_dump(old_values))
        old = render(chart=old_chart)
        args = one(old, "Deployment", "gameplane-operator")["spec"]["template"]["spec"]["containers"][0]["args"]
        assert not any(arg.startswith("--tunnel-resources=") for arg in args)


if __name__ == "__main__":
    main()
