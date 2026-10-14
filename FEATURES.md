# Gameplane features

Maturity labels: **stable** (shipped, on by default, and exercised by a bucketed e2e test), **beta** (shipped with caveats or without a bucketed e2e test), **experimental** (shipped but the docs say so), **planned** (not started), **in progress** (started, not yet merged to master).

Default column: **on** (enabled by the chart or the product), **off** (opt-in, ships disabled), or **n/a** (planned, in progress, or tooling with no install default).

Status date: 2026-10-14 (v0.3.0, pre-v1).

See [docs/roadmap.md](docs/roadmap.md) for what is planned and in progress.

## Servers and lifecycle

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Create a game server from a module template | stable | on | Creates a GameServer from an installed module template through the create wizard or the API. | [](README.md#features) |
| Start a stopped server | stable | on | Clears spec.suspend so the operator scales the pod back up. | [architecture](docs/architecture.md#start-a-server) |
| Stop a server | stable | on | Sets spec.suspend so the operator scales the pod to zero after the stop sequence. | [architecture](docs/architecture.md#start-a-server) |
| Restart a server | stable | on | Stamps a restart annotation so the pod restarts without changing desired state. | [](README.md#features) |
| Show server phase and conditions | stable | on | Shows Running, Starting, Stopping, Suspended, Failed and Pending with the reason conditions behind each. | [architecture](docs/architecture.md#start-a-server) |
| Clone a server without world data | stable | on | Copies the spec of a server into a new GameServer and leaves world data out. | [](README.md#features) |
| Switch game version on a running server | stable | on | Changes the template version of an existing server so the operator rolls it out. | [module-authoring](docs/module-authoring.md) |
| Delete a server | stable | on | Deletes a GameServer and its owned child objects. | [](README.md#features) |
| Transfer server ownership | stable | on | Lets the owner or an admin hand a server to another user. | [security](docs/security.md) |
| Manage server collaborators | stable | on | Lets the owner set collaborator users who can use start, stop, restart, wake and clone. | [security](docs/security.md) |
| Graceful stop with the game's native shutdown sequence | beta | on | Runs the template stop sequence over RCON, telnet or stdin before the pod scales to zero so world saves finish. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Wipe world data with typed confirmation | beta | on | Suspends the server, erases the world volume and resumes, for owners or admins only. | [architecture](docs/architecture.md) |
| Filter the server list by Running or Stopped | beta | on | Filters the server list with All, Running and Stopped tabs. | [unified-dashboard](docs/unified-dashboard.md) |
| List servers across all readable namespaces | beta | on | The Servers page lists servers from every namespace the viewer may read. | [unified-dashboard](docs/unified-dashboard.md) |
| Edit game configuration from the Settings tab | in progress | n/a | Lets an operator change a server game configuration from its Settings tab, with validation. | [roadmap](docs/roadmap.md#in-progress) |

## Console and admin actions

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Live console stream for PTY-mode games | stable | on | Attaches the browser to the container stdin and stdout through the pod attach API for templates with consoleMode pty. | [install](docs/install.md#pod-logs-and-pty-console-permissions) |
| Run template quick actions | stable | on | Runs template-defined admin actions such as save, broadcast, time, weather and reload from buttons. | [module-authoring](docs/module-authoring.md#actions) |
| Live game log tail | stable | on | Streams the game log file over a WebSocket from the agent sidecar. | [architecture](docs/architecture.md#tail-logs) |
| RCON command console | beta | on | Sends line-based commands to the game RCON protocol and shows responses as log lines. | [module-authoring](docs/module-authoring.md#rcon) |
| Console command history | beta | on | Recalls the last 100 sent commands with the arrow keys, kept only while the Console tab stays open. | [](README.md#features) |
| Clear, download and fullscreen console controls | beta | on | Toolbar buttons clear the terminal, download its buffer and toggle fullscreen without stopping the stream. | [](README.md#features) |
| Schema-validated action input with injection guard | beta | on | Validates action parameters against the template schema and escapes them before they reach the game console. | [dependencies](docs/dependencies.md#gameaction) |
| Stdin action transport for PTY-only games | beta | on | Runs actions on PTY-console games by writing lines to the container stdin through pods attach. | [module-authoring](docs/module-authoring.md#console-mode-and-game-log) |
| Kubernetes events timeline for a server | beta | on | Shows the 50 most recent Kubernetes events for the server pod and its parent objects. | [architecture](docs/architecture.md#start-a-server) |
| Container startup output stream | beta | on | Streams the game container stdout through the pod log API before the game log file exists. | [install](docs/install.md#pod-logs-and-pty-console-permissions) |
| Log replay on tab open | beta | on | Replays up to the last 500 game-log lines before following new output, capped by the agent at 20,000 lines and 4 MiB. | [architecture](docs/architecture.md#tail-logs) |
| Download the full game log | beta | on | Downloads the whole game log file without interrupting the live stream. | [architecture](docs/architecture.md#tail-logs) |

## Files

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Browse the exposed server filesystem | stable | on | Lists directories under the exposed data root with a breadcrumb. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Edit and save text files in the Monaco editor | stable | on | Opens a text file in the in-browser Monaco editor and saves it through the write endpoint. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Create a new text file | stable | on | Creates a new file in the open directory through the write endpoint. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Download a file | stable | on | Downloads one file from the server through the agent, including large files that stream past the request timeout. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Delete a file | stable | on | Removes one file through the delete endpoint after confirmation. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Create a folder | beta | on | Creates a folder in the open directory through the mkdir endpoint. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Upload files | beta | on | Uploads files into the open directory through the upload endpoint, writing them group-readable for the game process. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Delete a folder recursively | beta | on | Removes a folder and everything inside it after confirmation, and refuses folders that hold dot-prefixed entries. | [security](docs/security.md#agent-file-browser-path-confinement) |
| Confine file access to the exposed root | beta | on | Rejects path traversal and symlink escapes outside the exposed data root. | [security](docs/security.md#agent-file-browser-path-confinement) |

## Players

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| View online players and capacity | stable | on | Shows the online count, the maximum and the player list, refreshed every 5 seconds, for templates that declare a player source. | [](README.md#features) |
| View the whitelist and add or remove entries | stable | on | Lists whitelist entries and adds or removes names through the whitelist endpoints. | [](README.md#features) |
| Kick a player | beta | on | Disconnects an online player through the template kick command, and returns HTTP 501 when the template does not declare it. | [](README.md#features) |
| Ban and unban a player | beta | on | Adds or removes a persistent ban through the template ban and unban commands. | [](README.md#features) |
| View the banned players list | beta | on | Shows the banned players reported by the game through the banned endpoint. | [](README.md#features) |
| Steam display names on player lists | beta | off | Resolves Steam IDs to display names through the Steam Web API once a Secret is configured. | [install](docs/install.md#steam-display-names-optional) |

## Mods

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Install a mod from a download URL with registry metadata in the mod manifest | stable | on | Installs a mod file from a download URL and records its registry metadata in the per-volume mod manifest. | [](README.md#mod-management) |
| Upgrade an installed mod in place and remove it from the manifest | stable | on | Replaces an installed mod with a new file and removes the old file and its manifest entry, or removes the mod entirely. | [](README.md#mod-management) |
| Upload a mod file directly from the dashboard | stable | on | Accepts a multipart mod file upload for a server and installs it as a file-drop mod. | [](README.md#mod-management) |
| Reject path-traversal and symlink-escape mod archives | stable | on | Extracts valid mod archives while rejecting zip-slip paths and symlinks that escape the target directory. | [](README.md#mod-management) |
| Install mods by launch-parameter ID | beta | on | Projects selected mod IDs into the launch environment for games that fetch mods at startup. | [module-authoring](docs/module-authoring.md#mods) |
| Browse and install from Modrinth | beta | on | Searches and installs Modrinth mods scoped to the server game version and loader, with no key needed. | [](README.md#supported-registries) |
| Browse and install from Thunderstore | beta | on | Searches and installs Thunderstore packages for templates that set a community, with no key needed. | [](README.md#supported-registries) |
| Browse and install from Hangar | beta | on | Searches and installs Hangar plugins, with no key needed. | [](README.md#supported-registries) |
| Browse and install from the Factorio Mod Portal | beta | on | Installs Factorio mods from the mod portal using the user's factorio.com credentials entered in the install form. | [](README.md#supported-registries) |
| Browse and install from SpigotMC | beta | on | Searches and installs SpigotMC resources, with no key needed. | [](README.md#supported-registries) |
| Browse and install from GitHub Releases | beta | on | Installs release assets from a GitHub owner and repository set on the template, with no key needed. | [](README.md#supported-registries) |
| Browse and install from uMod | beta | on | Searches and installs uMod plugins, with no key needed. | [](README.md#supported-registries) |
| Store and replace registry API keys from Settings | beta | on | Saves, replaces and removes registry keys under Settings, Mod registries, kept in Kubernetes Secrets or the standalone database. | [](README.md#supported-registries) |
| Check installed mods for newer versions | beta | on | Reports available updates for mods installed from a provider that has a resolvable version. | [](README.md#mod-management) |
| Install a modpack from a provider | beta | on | Installs a modpack either by pinning it into a server env (env-mode) or by installing its dependency mods (deps-mode). | [module-authoring](docs/module-authoring.md#modpacks-providersmodpacks) |
| Browse and install from CurseForge | beta | off | Searches and installs CurseForge mods once a CurseForge API key is configured. | [](README.md#supported-registries) |
| Browse and install from Steam Workshop | beta | off | Searches and installs Steam Workshop items once a Steam Web API key is configured. | [](README.md#supported-registries) |
| Browse Nexus Mods | beta | off | Browses Nexus Mods once an API key is configured; downloads are not supported. | [](README.md#supported-registries) |

## Backups and restore

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Create an on-demand restic-snapshot backup of a server | stable | on | Starts a Backup that runs restic against the server data volume and records the snapshot ID. | [architecture](docs/architecture.md#back-up) |
| Connect a restic repository through a labelled Secret | stable | on | Reads the restic repository and credentials from a labelled Kubernetes Secret referenced by repoRef. | [architecture](docs/architecture.md#back-up) |
| Report a Failed backup for a missing PVC or bad credentials | stable | on | Sets a Backup to Failed with a message when its data volume is missing or its repository credentials are rejected. | [architecture](docs/architecture.md#back-up) |
| Delete a backup and forget its restic snapshot with prune | stable | on | Deleting a Backup runs restic forget with prune so the snapshot is removed from the repository. | [architecture](docs/architecture.md#backup-deletion-and-snapshot-cleanup) |
| Run backups on a cron schedule | stable | on | A BackupSchedule creates Backup resources on its cron expression and records last and next schedule times. | [architecture](docs/architecture.md#back-up) |
| Suspend a backup schedule | stable | on | Setting suspend on a BackupSchedule stops it from creating new backups. | [architecture](docs/architecture.md#back-up) |
| Trim old backups with keep-last and calendar retention rules | stable | on | Applies keepLast, hourly, daily, weekly, monthly and yearly retention after each scheduled backup; an empty policy keeps everything. | [architecture](docs/architecture.md#back-up) |
| Forbid overlapping scheduled backups | stable | on | A schedule with concurrencyPolicy Forbid does not start a new backup while one is still active. | [architecture](docs/architecture.md#back-up) |
| Restore a restic snapshot into an existing server | stable | on | Suspends the target GameServer, overwrites its data volume from a snapshot, and resumes it. | [architecture](docs/architecture.md#restore-from-backup) |
| Reject a restore whose source backup is missing or not Succeeded | stable | on | Sets a Restore to Failed when its Backup does not exist or has no successful snapshot. | [architecture](docs/architecture.md#restore-from-backup) |
| Quiesce game writes before each backup | beta | on | Asks the game to flush and pause writes before the snapshot when quiesce is true, which is the default on Backups. | [architecture](docs/architecture.md#back-up) |
| Tune backup and restore Job retry limit and deadline | beta | on | Sets the retry count and wall-clock deadline for restic backup and restore Jobs through Helm values. | [architecture](docs/architecture.md#back-up) |
| Browse backups and restores on the dashboard Backups page | beta | on | Lists per-server and global backups and restores with location, server and phase filters and a detail drawer. | [architecture](docs/architecture.md#back-up) |
| Create volume-snapshot CSI backups | beta | off | Captures a Kubernetes CSI VolumeSnapshot of the data volume instead of running restic, when a volume snapshot class is set. | [architecture](docs/architecture.md#back-up) |
| Restore a CSI volume-snapshot backup into a new GameServer | beta | off | Provisions a new GameServer seeded from a volume snapshot; the original server is not touched. | [architecture](docs/architecture.md#restore-from-backup) |
| Backup and restore drill runbook | planned | n/a | A documented runbook for periodic restore drills; the restore test exists but the runbook does not. | [roadmap](docs/roadmap.md#production-readiness-hardening-planned) |

## Game modules and catalog

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Discover modules from an OCI module source | stable | on | The operator indexes an OCI module source and lists its modules and versions in ModuleSource status. | [module-authoring](docs/module-authoring.md#module-sources) |
| Install a Module to materialize a GameTemplate | stable | on | Applying a Module CR makes the operator create the cluster-scoped GameTemplate for that module. | [module-authoring](docs/module-authoring.md#installing-a-module) |
| Block deleting a Module or GameTemplate that a server uses | stable | on | A finalizer keeps a Module or GameTemplate in place while a GameServer still uses its template. | [module-authoring](docs/module-authoring.md#installing-a-module) |
| Reject module source URLs that resolve to unsafe targets | stable | on | Module source fetches go through the SSRF dial guard, which refuses unsafe destinations. | [module-authoring](docs/module-authoring.md#module-sources) |
| Report an unreachable module registry on the Module | stable | on | A Module reports a failure when its source registry cannot be reached. | [module-authoring](docs/module-authoring.md#module-sources) |
| Upload module bundles to the upload source | stable | on | Bundles uploaded through the API are stored in an upload ModuleSource and indexed like any other source. | [module-authoring](docs/module-authoring.md#uploaded-bundles) |
| Switch a running server to another module version | stable | on | Changes a GameServer to another catalog version and loader, keeping each version and loader on its own data volume. | [module-authoring](docs/module-authoring.md#version-catalog-specversions) |
| Official game module catalog | beta | on | The official OCI catalog holds 29 game modules in the module repository, and the chart enables 16 of them by default. | [module-authoring](docs/module-authoring.md#module-sources) |
| Browse the merged module catalog with install and verification states | beta | on | The Modules page shows available, installed, verified, failed and update states across all configured sources. | [module-authoring](docs/module-authoring.md#installing-a-module) |
| Pin a Module to a bundle digest | beta | off | Sets Module spec digest so the operator installs only that exact bundle content. | [module-authoring](docs/module-authoring.md#verifying-and-pinning-bundles) |
| Verify cosign signatures on module bundles | beta | off | Checks a bundle signature against a configured public key before installing it. | [module-authoring](docs/module-authoring.md#verifying-and-pinning-bundles) |
| Verify keyless Sigstore identity signatures on module bundles | beta | off | Checks a bundle signed with a Fulcio identity and an optional transparency-log requirement. | [module-authoring](docs/module-authoring.md#verifying-and-pinning-bundles) |
| Use a Git repository as a module source | beta | off | Reads module directories from a Git repository at a pinned ref and optional subPath. | [module-authoring](docs/module-authoring.md#module-sources) |
| Use an HTTP archive as a module source | beta | off | Reads module directories from a tar.gz or zip archive served over HTTP or HTTPS. | [module-authoring](docs/module-authoring.md#module-sources) |
| Use a local directory as a module source | beta | off | Reads module directories from a path mounted into the operator pod. | [module-authoring](docs/module-authoring.md#module-sources) |
| Complete specs.md for all 29 game modules | planned | n/a | Each game module carries a specs.md documenting its template configuration; 26 of 29 have one today. | [roadmap](docs/roadmap.md#game-module-template-specifications-planned) |
| Nuclear Option join through a load-balancer IP pool override | in progress | n/a | Lets Nuclear Option clients join through a load-balancer IP pool override; blocked until the server binding on UDP 7777 is documented. | [roadmap](docs/roadmap.md#in-progress) |

## Module authoring

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Declare a mods capability in a GameTemplate | stable | on | A template declares file-drop paths or an ID list so the mod install flow knows where mods go. | [module-authoring](docs/module-authoring.md#mods) |
| Render an install wizard from a config schema | beta | on | A template config schema generates the install and settings form for a server. | [module-authoring](docs/module-authoring.md#config-schema--wizard) |
| Declare admin actions with typed inputs | beta | on | A template lists admin actions with input definitions that the console and API render and validate. | [module-authoring](docs/module-authoring.md#actions) |
| Author modules in the dashboard Modules builder | beta | on | The dashboard offers a builder that creates module content without the CLI. | [module-authoring](docs/module-authoring.md#5-web-dashboard-module-builder) |
| Scaffold a new module with gp-module init | beta | n/a | Generates a module directory with module.yaml, template.yaml and README.md for a chosen archetype. | [module-authoring](docs/module-authoring.md#1-scaffolding-a-new-module-init) |
| Validate a module offline with gp-module validate | beta | n/a | Checks a module directory offline and can emit machine-readable JSON for CI. | [module-authoring](docs/module-authoring.md#2-instant-offline-validation-validate) |
| Dry-run a template config with gp-module preview | beta | n/a | Shows the runtime config a template would produce for a given memory limit and user configuration. | [module-authoring](docs/module-authoring.md#3-dry-run-config-preview-preview) |
| Package a module bundle with gp-module package | beta | n/a | Builds a local tar.gz bundle from a module directory, with package size limits applied. | [module-authoring](docs/module-authoring.md#4-packaging-and-distribution-package) |
| Push a module bundle to an OCI registry | beta | n/a | Publishes a packaged module bundle as an OCI artifact for a module source to index. | [module-authoring](docs/module-authoring.md#pushing-a-bundle) |
| Sign module bundles with cosign in the release pipeline | beta | n/a | Signs official bundles with a cosign key held in CI secrets so verifying sources can check them. | [module-authoring](docs/module-authoring.md#signing-official-bundles) |
| Editor JSON schemas for module.yaml and template.yaml | beta | n/a | Provides yaml-language-server schemas so editors validate and autocomplete module and template files. | [module-authoring](docs/module-authoring.md#editor-schemas) |

## Networking and tunnels

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Expose a server through a ClusterIP Service | stable | on | The default expose mode fronts the game pod with a Service reachable only inside the cluster. | [networking](docs/networking.md) |
| Apply a per-server egress NetworkPolicy for the tunnel pod | beta | on | Creates an egress policy for a tunnel-enabled server that admits DNS, advertised ports and provider relay ports. | [tunnels](docs/tunnels.md#troubleshooting) |
| Expose a server through a NodePort Service | beta | off | An opt-in expose mode publishes game ports on node ports, with an optional pinned port number. | [tunnels](docs/tunnels.md#interaction-with-other-networking-fields) |
| Expose a server through a LoadBalancer Service | beta | off | An opt-in expose mode asks the cluster load-balancer address manager for an external address. | [networking](docs/networking.md#address-pool-overview) |
| Expose a server through a Hostport mapping | beta | off | An opt-in expose mode binds game ports on the node host, which the wake sentinel cannot hold. | [roadmap](docs/roadmap.md) |
| Request a named address pool for a server | beta | off | Writes a MetalLB or Cilium pool preference onto a LoadBalancer Service, recorded as unhonored when no address manager is set. | [networking](docs/networking.md#setting-an-address-pool-or-specific-address) |
| Request a specific external IP address for a server | beta | off | Sets spec.networking.address so the address manager assigns that IP. | [networking](docs/networking.md#setting-an-address-pool-or-specific-address) |
| Report address assignment progress on the AddressAssignment condition | beta | off | Shows Assigned, AssignmentPending, ServiceNotReady, IgnoredForExposureMode and other reasons on the server status. | [networking](docs/networking.md#troubleshooting-address-assignment) |
| Refuse an address already held by another GameServer | beta | off | Reports AddressInUse when another GameServer in the cluster already reports the same address. | [networking](docs/networking.md#troubleshooting-address-assignment) |
| Report a missing or exhausted address pool | beta | off | Reports PoolNotFound or AllocationFailed when the requested pool is absent or has no free addresses. | [networking](docs/networking.md#troubleshooting-address-assignment) |
| Change the address pool of a running server | beta | off | Updates pool metadata on a running server, while whether the IP changes depends on the load balancer. | [networking](docs/networking.md#live-address-changes) |
| Relay players through an frp tunnel | beta | off | Runs an frpc tunnel pod that dials your own frps server and advertises serverAddr:remotePort. | [tunnels](docs/tunnels.md#frp) |
| Relay players through a Tailscale tailnet | beta | off | Joins the game pod to your tailnet with an auth key so only tailnet devices can reach it. | [tunnels](docs/tunnels.md#tailscale) |
| Relay players through a playit.gg tunnel | beta | off | Runs a playit tunnel and reports its assigned public address into status.endpoints, rechecked every 30 seconds. | [tunnels](docs/tunnels.md#playitgg) |
| Store tunnel credentials as owned Secrets | beta | off | Accepts relay credentials from the dashboard or the tunnel-credentials API and refuses Secrets without a matching ownerReference. | [tunnels](docs/tunnels.md#security) |
| Keep the tunnel pod running while the game sleeps | beta | off | The tunnel Deployment keeps the relay connection and public address up while the game pod is asleep. | [tunnels](docs/tunnels.md#interaction-with-idle-auto-sleep-and-wake-on-connect) |
| Restrict game ingress with a sourceRanges allowlist | beta | off | Limits Service-port traffic to listed source ranges, and does not gate traffic arriving through a tunnel. | [tunnels](docs/tunnels.md#interaction-with-other-networking-fields) |
| Report conflicting networking options as a status condition | planned | n/a | Would flag redundant settings such as a public LoadBalancer plus a tunnel as an informational condition. | [roadmap](docs/roadmap.md) |

## Idle sleep and wake

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Wake a sleeping server with the wake verb | beta | on | POST /servers/{name}:wake clears the operator sleep marker so the game pod comes back. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Show slept servers as Asleep with a Wake button in the dashboard | beta | on | The dashboard labels a slept server Asleep and offers Wake instead of Start. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Opt in to idle auto-sleep after a set number of empty minutes | beta | off | Opt-in spec.idle lets the operator scale a server to zero after it reports no players for afterMinutes, with a 5-minute floor. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Sleep an idle server through the module stop sequence | beta | off | Sleep runs the same softStop path as a manual stop, then reads Suspended with an IdleAsleep reason. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Wake a sleeping server on a cron wake window | beta | off | spec.idle.wakeWindows cron expressions in the cluster timezone clear the sleep marker at each tick. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Show why a server will not sleep | beta | off | status.idle.reason and the dashboard explain when idle time is not accruing. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Freeze the idle clock on stale heartbeats or missing player counts | beta | off | A stale agent heartbeat or absent playersOnline pauses idle time, so unknown is never read as empty. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Report idle sleep state in the gameplane_gameservers_idle metric | beta | off | An operator gauge labelled asleep or awake for idle-enabled servers. | [install](docs/install.md#metrics) |
| Arm wake-on-connect for a server | beta | off | spec.idle.wakeOnConnect makes the operator run a sentinel pod that holds advertised ports while the game is asleep. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Wake on a Minecraft join but not on a server-list ping | beta | off | The Minecraft parser answers status pings in place and wakes the game only for a real join. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Wake on a Terraria join through the Terraria parser | beta | off | Terraria and tModLoader modules declare the Terraria handshake parser. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Wake other games with a packets-in-window heuristic | beta | off | Games without a parser wake on traffic and drop the first client, which must retry. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Send a try-again disconnect when a held join misses the deadline | beta | off | Minecraft and Terraria clients receive a protocol-native waking-up message after the 25-second default deadline. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Choose a wake protocol per game port | beta | off | GamePort.wakeProtocol selects minecraft, terraria, generic or none for each advertised port. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Hold advertised ports by switching Service endpoints during sleep | beta | off | While asleep the Service points at the sentinel, and on wake it points back at the game pod. | [architecture](docs/architecture.md#sleep-an-idle-server) |
| Bounce held clients on Hostport servers instead of holding them | beta | off | Hostport cannot be held by the sentinel, so it bounces clients and exits on wake. | [architecture](docs/architecture.md#sleep-an-idle-server) |

## Multi-cluster

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Register a remote cluster with its kubeconfig | stable | on | The Clusters page or POST /clusters stores a remote Kubernetes target that requests are routed to. | [multicluster-ui](docs/multicluster-ui.md#registered-clusters) |
| Create a server on a chosen registered cluster | stable | on | A create request with ?cluster=<id> places the GameServer on that cluster's own Kubernetes API. | [unified-dashboard](docs/unified-dashboard.md#user-experience) |
| Read servers from a selected cluster with ?cluster= dispatch | stable | on | Listing and reads resolve the requested cluster, reject unknown cluster ids with 400, and keep remote servers out of the local listing. | [unified-dashboard](docs/unified-dashboard.md#data-and-authorization-contract) |
| Enforce per-cluster RBAC grants on remote servers | stable | on | A viewer bound only to the local cluster is forbidden from reading remote servers. | [multicluster-ui](docs/multicluster-ui.md#user-access) |
| Refuse fallback to a same-named local server when a remote target is missing | stable | on | A missing remote server never resolves to the home namesake, including registry, modpack and capture routes. | [remote-parity](docs/remote-parity.md#routing-and-identity) |
| Keep node join tokens and kubeconfig issuance local to the central cluster | stable | on | Node join token and kubeconfig endpoints are available only for the local cluster. | [multicluster-ui](docs/multicluster-ui.md#node-inventory) |
| Remove a registered cluster and its kubeconfig Secret | stable | on | Removing a registration deletes the kubeconfig Secret the API created for it. | [multicluster-ui](docs/multicluster-ui.md#registered-clusters) |
| Rotate a registered kubeconfig in place | beta | on | PUT /clusters/{name}/kubeconfig replaces the credential, keeps identity and grants, and reloads the client. | [standalone-panel](docs/standalone-panel.md#rotate-a-registered-kubeconfig) |
| Show Kubernetes API health per registered cluster | beta | on | The Clusters page shows API health, version and last check, but not gateway or game health. | [multicluster-ui](docs/multicluster-ui.md#registered-clusters) |
| Browse node inventory for a selected cluster | beta | on | The Cluster page lists nodes and capacity for the chosen cluster, and a failed remote read shows an error rather than local data. | [multicluster-ui](docs/multicluster-ui.md#node-inventory) |
| Show combined server and player counts across authorized sites | beta | on | The dashboard totals authorized sites and labels unavailable or truncated scopes next to partial totals. | [unified-dashboard](docs/unified-dashboard.md#user-experience) |
| Filter servers, backups and schedules by location | beta | on | Location filters in the existing popovers narrow combined lists without a global cluster switcher. | [unified-dashboard](docs/unified-dashboard.md#filter-layout) |
| Search servers across sites without identity collisions | beta | on | One search covers authorized servers from every site, keeping cluster, namespace and name on each result. | [unified-dashboard](docs/unified-dashboard.md#user-experience) |
| Open canonical server links that carry cluster and namespace | beta | on | Direct links and fresh tabs target the named cluster and namespace, ignoring stored cluster selections. | [unified-dashboard](docs/unified-dashboard.md#acceptance) |
| Route registry, modpack and mod-ID writes to the selected cluster | beta | on | Mod and modpack configuration writes go to the selected cluster with resource-version and UID protection. | [remote-parity](docs/remote-parity.md#routing-and-identity) |
| Block unsafe address classes for remote Kubernetes and gateway dials | beta | on | Dial-time checks reject loopback, link-local, metadata, unspecified and multicast destinations. | [standalone-panel](docs/standalone-panel.md#install-and-register-remote-game-clusters) |
| Run remote agent operations through the gateway | beta | off | The optional private gateway carries agent operations such as files, mods and player data to a registered cluster over mTLS. | [multicluster-agent-gateway](docs/multicluster-agent-gateway.md#routing-and-identity) |
| Register gateway URL and mTLS credentials per cluster | beta | off | Cluster.spec.agentGateway (combined) or PUT /clusters/{name}/gateway (standalone) stores the gateway endpoint and labelled credentials. | [multicluster-agent-gateway](docs/multicluster-agent-gateway.md#register-the-gateway-with-the-central-api) |
| Check gateway capabilities and refuse older agents | beta | off | The gateway advertises protocol and capture support, and agents without the UID route fail closed. | [multicluster-agent-gateway](docs/multicluster-agent-gateway.md#routing-and-identity) |
| Download and delete remote capture files through the gateway | beta | off | Capture files are fetched and removed over a UID-bound gateway route, and unbound older files fail closed. | [multicluster-agent-gateway](docs/multicluster-agent-gateway.md#failure-and-rotation-behavior) |
| Rotate gateway trust and revoke gateway access | beta | off | Repeated PUT rotates gateway credentials, and DELETE removes gateway access while keeping the cluster registration. | [standalone-panel](docs/standalone-panel.md#configure-remote-agent-access) |
| Keep games running when the gateway is down | beta | off | A gateway outage stops remote agent access but not games or the local operator. | [multicluster-agent-gateway](docs/multicluster-agent-gateway.md#failure-and-rotation-behavior) |
| Run the central panel without a local Kubernetes cluster | beta | off | api.standalone runs the dashboard and API with no operator, and the central host never runs games. | [standalone-panel](docs/standalone-panel.md#central-panel-on-kubernetes) |
| Install the standalone panel with Docker Compose | beta | off | deploy/standalone/compose.yaml runs the API and dashboard without a Kubernetes install, bound to 127.0.0.1:8080. | [standalone-panel](docs/standalone-panel.md#docker-compose) |
| Store standalone remote credentials encrypted with a persistent panel key | beta | off | Remote credentials are encrypted in SQL with a key file that must be backed up with the database. | [standalone-panel](docs/standalone-panel.md#storage-and-recovery) |
| Rotate the panel master encryption key offline | beta | off | rotate-panel-key re-encrypts management credentials under a new key file in one transaction. | [standalone-panel](docs/standalone-panel.md#rotate-the-master-encryption-key) |
| Run standalone with a provisioned panel key that is never generated | beta | off | The provisioned-key option requires an existing 32-byte key and refuses to create one. | [standalone-panel](docs/standalone-panel.md#provision-a-key-outside-the-database-storage) |
| Restrict remote destinations with an allowed CIDR list | beta | off | GAMEPLANE_REMOTE_ALLOWED_CIDRS limits Kubernetes API and gateway destinations to the listed networks. | [standalone-panel](docs/standalone-panel.md#install-and-register-remote-game-clusters) |
| Select a remote cluster for module installs and sources in standalone panels | beta | off | Standalone modules and module sources act on an explicitly selected registered cluster. | [standalone-panel](docs/standalone-panel.md#grant-workload-access) |

## Users, auth and access

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Sign in with local username and password accounts | stable | on | Local accounts sign in with argon2id-hashed passwords and receive an HttpOnly, Secure, SameSite=Lax session cookie. | [security](docs/security.md#authentication) |
| Log out and end the session on the server | stable | on | Logging out invalidates the session so the old cookie no longer authenticates. | [security](docs/security.md#authentication) |
| End existing sessions on dashboard password reset | stable | on | A dashboard password reset invalidates every existing session of that account. | [security](docs/security.md#authentication) |
| Throttle login attempts per client IP address | stable | on | Wrong-password login attempts are rate limited per source IP and return 429 once the bucket is exhausted. | [security](docs/security.md#client-ip-extraction-from-forwarded-headers) |
| Show generic login errors with no cluster or account details | stable | on | The login page and other unauthenticated views show only generic messages such as invalid credentials. | [security](docs/security.md#pre-auth-screens) |
| Disable or re-enable the local sign-in provider from auth settings | stable | on | Admins can disable local login from Authentication settings, after which the login endpoint refuses with a neutral response until it is re-enabled. | [security](docs/security.md#dashboard-managed-providers) |
| Check every API route against a fixed permission catalog | stable | on | Each route maps to one resource:action permission from the server-defined catalog, and unmatched routes fail closed. | [security](docs/security.md#authorization) |
| Use the built-in admin, operator and viewer roles | stable | on | Three seeded roles give admin full access, operators server control without user management, and viewers read-only access. | [security](docs/security.md#authorization) |
| Create custom roles from permission catalog subsets | stable | on | Admins create, edit and delete custom roles holding any subset of the catalog, never the wildcard. | [security](docs/security.md#authorization) |
| Keep built-in roles immutable and block deleting assigned roles | stable | on | The admin role is immutable, built-in roles cannot be deleted, and any role still assigned to a user cannot be deleted. | [security](docs/security.md#authorization) |
| Bind a role to a user within one namespace | stable | on | Per-namespace role bindings grant a role in a single namespace, while the wildcard namespace gives cluster-wide access. | [security](docs/security.md#authorization) |
| Restrict destructive server actions to the owner or an admin | stable | on | Delete, wipe, ownership transfer and collaborator edits require the server's owner or an admin. | [security](docs/security.md#per-gameserver-access-owner--collaborators) |
| Remove a user account with all linked data | stable | on | Deleting a user removes SSO links, preferences, sessions, API tokens and role bindings and revokes the share links that account created. | [security](docs/security.md#authorization) |
| Save personal dashboard theme preferences | stable | on | Any signed-in user can read and save their own dashboard theme preferences. | [roadmap](docs/roadmap.md#shipped-in-v030) |
| Require the CSRF header on mutating requests | beta | on | Mutating requests must echo the CSRF cookie value in the X-Gameplane-CSRF header (double-submit). | [security](docs/security.md#authentication) |
| Refuse an auth config save that leaves no sign-in provider enabled | beta | on | The API rejects an auth config save with zero enabled providers, counting the Helm provider as always enabled. | [security](docs/security.md#lockout-guard-and-break-glass) |
| Refuse to demote or delete the last user manager | beta | on | The API refuses to demote or delete the last user who can manage users and refuses self-demotion below users:manage. | [security](docs/security.md#authorization) |
| Require a cluster-wide binding for cluster-scoped permissions | beta | on | Cluster-scoped permissions such as users:manage need a cluster-wide binding, the same Role versus ClusterRole split Kubernetes uses. | [security](docs/security.md#authorization) |
| Create and edit dashboard user accounts | beta | on | Admins create users, change a user's primary role and manage per-namespace bindings from the Users page. | [security](docs/security.md#authorization) |
| Share a read-only status link for one GameServer | beta | on | A token in a share link gives unauthenticated access to one GameServer's status and connection address, optionally with permission to wake it. | [security](docs/security.md#share-links) |
| Reset an admin password from the API pod with bootstrap-admin | beta | n/a | Running bootstrap-admin with --force resets a named account's password, promotes it to admin and ends its existing sessions. | [security](docs/security.md#lockout-guard-and-break-glass) |
| Re-enable local login from the API pod with bootstrap-admin | beta | n/a | Running bootstrap-admin with --enable-local-login force-enables the local provider so a locked-out admin can sign in. | [security](docs/security.md#lockout-guard-and-break-glass) |
| Sign in with dashboard-managed OIDC providers | beta | off | Admins add OIDC providers from Authentication settings, store the client secret in a labelled Secret, and apply changes without restart. | [security](docs/security.md#dashboard-managed-providers) |
| Sign in with the Helm-configured OIDC provider | beta | off | The api.oidc Helm values add a read-only helm provider that the dashboard cannot edit, disable or delete. | [security](docs/security.md#dashboard-managed-providers) |
| Follow OIDC setup guides for five identity providers | beta | n/a | Step-by-step guides cover registering Gameplane as an OIDC client in Keycloak, Authentik, Google, Okta and Entra ID. | [oidc](docs/oidc.md#provider-guides) |
| Assign roles from IdP group claims at install time | beta | off | Helm-seeded group mappings for admin, operator and viewer assign the most privileged matching role, with a default role when nothing matches. | [oidc](docs/oidc.md#role-mapping-at-install-time-no-bootstrap-admin-required) |
| Re-evaluate the OIDC role on every login | beta | off | The role is recomputed from current IdP groups at each OIDC login, so a removed group lowers access on the next sign-in. | [security](docs/security.md#re-evaluation-on-every-login) |
| Override a role's IdP group list from the dashboard | beta | off | Admins replace the group list for a role in Authentication settings, effective at the next login, and reset it to the Helm value. | [oidc](docs/oidc.md#managing-role-mappings-from-the-dashboard) |
| Service accounts on the Users page | planned | n/a | A Users page tab for service accounts is planned for v1.1. | [roadmap](docs/roadmap.md) |

## Audit and security

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Record every mutating API request in the audit log | stable | on | Each mutating API request writes an audit event to the audit_events table that the dashboard serves. | [install](docs/install.md#audit-log) |
| Page and filter the audit log | stable | on | The audit view pages through events and applies filters, backed by GET /admin/audit. | [install](docs/install.md#audit-log) |
| Encrypt agent and capture control traffic with mTLS | stable | on | Operator-managed agent and capture-sidecar control connections use mutual TLS 1.2 or newer with a verified client certificate. | [security](docs/security.md#authentication) |
| Isolate pods with NetworkPolicies | stable | on | NetworkPolicies restrict game, API and backup traffic, including default-deny egress for game pods. | [security](docs/security.md#networkpolicies) |
| Export the audit trail as CSV or JSON | beta | on | GET /admin/audit/export downloads the full audit trail for offline review. | [install](docs/install.md#audit-log) |
| Verify the audit hash chain for tampering | beta | on | GET /admin/audit/verify re-walks the SHA-256 chain and reports the first broken link, catching naive in-database edits and tail truncation. | [security](docs/security.md#audit-log-integrity) |
| Resolve client IPs through trusted proxies | beta | on | The API reads X-Forwarded-For only from peers inside api.trustedProxies, so audit records and per-IP limits use the real client address. | [security](docs/security.md#client-ip-extraction-from-forwarded-headers) |
| Run Gameplane pods with hardened security contexts | beta | on | Operator, API, agent and optional bridge pods run as uid 65532 with read-only root filesystems, RuntimeDefault seccomp and all capabilities dropped. | [security](docs/security.md#pod-security) |
| Redact GameServer password values in API responses | beta | on | Every API response carrying a GameServer replaces non-empty password values with a redaction marker and fails closed if the template cannot be read. | [security](docs/security.md#gameserver-config-passwords) |
| Confine dashboard-managed credentials to labelled Secrets | beta | on | The API reads and deletes only Secrets carrying its own labels for auth providers and notification sinks, so a config write cannot aim at arbitrary Secrets. | [security](docs/security.md#dashboard-managed-providers) |
| Prune audit events past a retention age | beta | off | api.audit.retentionDays removes events older than N days, and the default of 0 keeps events forever. | [install](docs/install.md#audit-log) |
| Mirror audit events to stdout | beta | off | api.audit.stdout emits each audit event as a structured JSON log line for a cluster log aggregator. | [install](docs/install.md#audit-log) |
| Forward audit events to a webhook | beta | off | api.audit.webhook.url POSTs each event as JSON from a bounded best-effort buffer, with an optional Authorization header read from a Secret. | [install](docs/install.md#audit-log) |
| Deploy the bundled audit-syslog-bridge from Helm | beta | off | api.audit.webhook.syslogBridge.enabled deploys the bridge and points the audit webhook at it automatically. | [audit-syslog-bridge](audit-syslog-bridge/README.md#run-via-the-helm-chart) |
| Deliver audit events to syslog over TCP, TLS or UDP | beta | off | audit-syslog-bridge turns each audit JSON document into an RFC 5424 syslog record and sends it over TCP, TCP with TLS or UDP, with TCP recommended. | [audit-syslog-bridge](audit-syslog-bridge/README.md#transport-prefer-tcp) |
| Stream audit batches to S3-compatible storage | beta | off | api.audit.s3 batches audit events as NDJSON objects, flushed at 100 events, 1 MiB or 5 seconds, to AWS S3, MinIO or a similar endpoint. | [install](docs/install.md#audit-log) |
| Audit OIDC role assignments and mapping overrides | beta | off | Each OIDC role assignment, override and reset writes an audit event naming the provider, matched group and role transition. | [security](docs/security.md#audit-trail-every-assignment-override-and-reset) |
| Enforce restricted Pod Security on the games namespace | beta | off | podSecurity.enforceRestricted applies the restricted Pod Security Standard to the games namespace. | [security](docs/security.md#pod-security) |
| Verify release signatures with the current cosign key | beta | n/a | The ECDSA P-256 cosign key verifies releases from the July 2026 rotation onward, and the retired Ed25519 key verifies earlier releases. | [key-rotation](docs/key-rotation.md#verifying-artifacts) |
| Sign the audit chain with a keyed HMAC | planned | n/a | An HMAC-keyed audit chain with the key held outside the database is a candidate future hardening. | [roadmap](docs/roadmap.md) |

## Notifications

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Store sink credentials in API-created Secrets | beta | on | Entering a sink's URL, ntfy topic or SMTP settings stores them in a Secret the API creates and labels, linked through configRef. | [notifications](docs/notifications.md#sink-credentials-secrets) |
| Alert when a server becomes unhealthy or recovers | beta | on | server.unhealthy fires when a GameServer fails or loses its heartbeat, and server.recovered fires when it becomes healthy again. | [notifications](docs/notifications.md#events) |
| Alert when a backup or restore fails | beta | on | backup.failed and restore.failed fire when a Backup or Restore enters the Failed phase. | [notifications](docs/notifications.md#events) |
| Send a test event to a saved sink | beta | on | A POST to the sink test endpoint delivers a synthetic event synchronously and returns the real delivery outcome. | [notifications](docs/notifications.md#testing-a-sink) |
| Retry transient delivery failures | beta | on | Network errors and 5xx responses are retried twice with 2 and 8 second backoffs, while 4xx responses are not retried. | [notifications](docs/notifications.md#delivery-semantics-and-observability) |
| Count deliveries in Prometheus metrics | beta | on | gameplane_notify_deliveries_total counts sent, failed, dropped and skipped deliveries per sink kind. | [notifications](docs/notifications.md#delivery-semantics-and-observability) |
| Deliver notifications without blocking reconciliation or requests | beta | on | Notifications are sent from a bounded in-memory queue, so a slow or down sink never blocks reconciliation or an API request. | [notifications](docs/notifications.md#delivery-semantics-and-observability) |
| Guard sink dials with the SSRF policy | beta | on | Every sink dial, over HTTP or SMTP, goes through the netguard IsAllowed policy and refuses link-local, unspecified, multicast and NAT64 destinations at dial time. | [security](docs/security.md#notifications) |
| Keep sink URLs out of delivery errors | beta | on | Delivery errors are sanitized so the sink URL, which often embeds a capability token, never appears in them. | [security](docs/security.md#notifications) |
| Label alerts with the install's instance name | beta | on | The Admin Settings instance name is sent as the instance field so alerts from different installs can be told apart. | [notifications](docs/notifications.md#payloads) |
| Post structured JSON webhook payloads | beta | on | The generic webhook receives a JSON object with type, ts, kind, name, namespace, message, instance and test fields. | [notifications](docs/notifications.md#payloads) |
| Send alerts to Discord, Slack, ntfy, SMTP or a generic webhook | beta | off | Admins add notification sinks in Admin Settings, and each sink kind receives its own payload format. | [notifications](docs/notifications.md#sink-credentials-secrets) |
| Use pre-created labelled Secrets for sinks | beta | n/a | A Secret made with kubectl or GitOps works when it carries the gameplane.local/notification-sink label and a matching configRef. | [notifications](docs/notifications.md#sink-credentials-secrets) |
| Alert when a backup or restore succeeds | beta | off | backup.succeeded and restore.succeeded are opt-in events that a sink must select. | [notifications](docs/notifications.md#events) |
| Send email alerts over SMTP with selectable TLS | beta | off | The smtp sink takes host, from, to and optional port, username, password and a TLS mode of starttls (default), implicit or none. | [notifications](docs/notifications.md#sink-credentials-secrets) |

## Telemetry

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Send basic usage counts once a day | stable | on | The API sends its version, GameServer count and GameTemplate count once a day, with no names, namespaces or addresses. | [install](docs/install.md#telemetry) |
| Hold reports until an admin sees the first-login notice | stable | on | On a new install no report is sent until the first admin has seen the notice listing both tiers and the destination. | [install](docs/install.md#telemetry) |
| Turn basic telemetry off from Admin Settings | stable | on | Admin Settings telemetry toggles stop reports, and turning them back on resumes reporting without sending early. | [install](docs/install.md#telemetry) |
| Hard-disable telemetry at install time | stable | on | Setting api.telemetry.enabled to false stops all sending and hides the first-login notice, overriding the admin toggles. | [install](docs/install.md#telemetry) |
| Send extended usage reports with a random install ID | stable | on | The extended tier adds a random install ID, environment bands, per-module counts, feature flags and the UI language, all from fixed sets. | [install](docs/install.md#telemetry) |
| Send reports to the project default receiver | beta | on | An empty api.telemetry.endpoint sends reports to the project default receiver at telemetry.gameplane.net, and a custom endpoint replaces it. | [install](docs/install.md#telemetry) |
| Reset the install ID from Admin Settings | beta | off | An admin resets the pseudonymous install ID, and the install then looks new to the receiver. | [install](docs/install.md#telemetry) |
| Sign extended reports with an install key | beta | off | Extended reports carry an Ed25519 signature, so forged, unsigned, tampered or replayed reports are refused without changing stored data. | [security](docs/security.md#signing-what-it-does-and-doesnt-stop) |
| Run a bundled telemetry receiver from Helm | beta | off | api.telemetry.receiver.enabled deploys a receiver beside the install, with a PVC for its SQLite data. | [telemetry-provider](docs/telemetry-provider.md#deploy-on-kubernetes) |
| Validate and aggregate incoming reports strictly | beta | off | The receiver decodes each report strictly, folds it into daily aggregates and expiring activity records, and never stores raw reports or source addresses. | [telemetry-receiver](telemetry-receiver/README.md#behavior) |
| Limit accepted reports per source per day | beta | off | INGEST_SOURCE_DAILY_LIMIT caps accepted reports per source per UTC day, with a default of 20. | [telemetry-provider](docs/telemetry-provider.md#configuration) |
| Publish a public summary of five counts | beta | off | GET /v1/summary returns five fixed counts without credentials when PUBLIC_SUMMARY is true. | [telemetry-provider](docs/telemetry-provider.md#public-summary) |
| Browse the receiver dashboard behind a token | beta | off | The receiver dashboard on port 8081 starts only when DASHBOARD_TOKEN is at least 32 characters, and unauthenticated responses reveal nothing. | [security](docs/security.md#dashboard-authentication-and-the-refusal-invariant) |
| Scrape receiver metrics with a bearer token | beta | off | The receiver's /metrics on the dashboard listener accepts only the Bearer token and is absent from the public port. | [telemetry-provider](docs/telemetry-provider.md#scrape-metrics-with-the-token) |
| Require proof of work on report ingestion | beta | off | INGEST_POW makes reporters solve a challenge whose difficulty rises with request rate, to deter bulk fabricated reports. | [telemetry-provider](docs/telemetry-provider.md#proof-of-work-on-ingest) |
| Expire receiver data on a retention schedule | beta | off | RETENTION_DAYS (default 730) and ACTIVITY_EXPIRY_DAYS (default 90) remove old aggregates and activity records on an hourly sweep. | [telemetry-provider](docs/telemetry-provider.md#retention) |
| Pseudonymise install IDs with a pepper | beta | off | The receiver stores only an HMAC of each install ID keyed with ID_PEPPER, generated and kept in its database if unset. | [telemetry-provider](docs/telemetry-provider.md#secrets) |
| Run a standalone telemetry provider | beta | n/a | A runbook covers deploying the receiver image, TLS, secrets, Cloudflare settings, metrics and backups for a provider serving other installs. | [telemetry-provider](docs/telemetry-provider.md) |
| Show a version-trend view in the receiver dashboard | in progress | n/a | A version-trend view in the telemetry dashboard is in progress under the spec 022 amendment T101 to T110. | [roadmap](docs/roadmap.md) |

## Observability and admin

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| API Prometheus metrics on a private listener | stable | on | The API serves Prometheus metrics on a dedicated listener that the public port does not expose. | [install](docs/install.md#metrics) |
| Operator controller and fleet gauges | beta | on | The operator exports controller-runtime metrics and the gameplane_gameservers and gameplane_backups phase gauges. | [install](docs/install.md#metrics) |
| Per-server agent metrics | beta | on | Each game pod agent exposes gameplane_agent CPU, memory, disk and player gauges on a plain metrics port 9090. | [install](docs/install.md#metrics) |
| Admin system log viewer | beta | on | Admins follow API and operator logs in separate streams, filter them, and download raw records. | [install](docs/install.md#observability) |
| ServiceMonitor and PodMonitor scrape objects | beta | off | Helm can create Prometheus Operator objects that scrape the operator, API, telemetry receiver and agents. | [install](docs/install.md#observability) |
| Operator PrometheusRule alerts | beta | off | Helm can ship operator reconcile, workqueue, GameServer Failed and Backup Failed alert rules. | [install](docs/install.md#alerts) |
| Operator Grafana dashboard | beta | off | Helm can ship a Grafana operator dashboard as a ConfigMap for sidecar auto-import. | [install](docs/install.md#observability) |
| Packet capture from game pods | beta | off | Admins start opt-in, BPF-filtered PCAPNG captures on a running game pod and download the files. | [install](docs/install.md#values-reference) |
| Automatic deletion of expired captures | beta | off | Completed packet captures are deleted automatically after their retention window ends. | [install](docs/install.md#values-reference) |

## Localization

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Localized dashboard and community language packs | in progress | n/a | Spec 019 adds app localization and community language packs; the work is on open PR #668 and is not yet on master. | [roadmap](docs/roadmap.md#in-progress) |

## Developer and operator tooling

| Capability | Maturity | Default | Description | Docs |
|---|---|---|---|---|
| Install the panel from the Helm chart | stable | on | One helm upgrade --install from the GHCR OCI registry installs the operator, API and dashboard. | [install](docs/install.md#one-shot-install) |
| CRD auto-apply on helm upgrade | stable | on | A pre-upgrade hook applies the Gameplane CRDs so every helm upgrade keeps the schema current. | [install](docs/install.md#helm-crd-caveat) |
| Read-only MCP server | beta | off | An optional MCP server lets AI assistants read GameServers, pods, events and logs and propose fixes without writing. | [mcp-server](mcp-server/README.md#tools) |
| Helper image overrides for air-gapped mirrors | beta | n/a | Helm lets operators retag the config-init and restic helper images to a private registry mirror. | [install](docs/install.md#values-reference) |
| Rolling edge images | beta | off | Setting image.tag to edge runs the rolling build published on every master push. | [install](docs/install.md#edge-channel-latest-edge-build) |
| Local Kind development loop | beta | n/a | Make targets create a Kind cluster with a local OCI registry, load images and run the web dev server. | [contributing](docs/contributing.md#dev-loop) |
| Store API data in PostgreSQL | experimental | off | Runs the API against a PostgreSQL database instead of SQLite, but only in images built with the postgres build tag. | [install](docs/install.md#values-reference) |
