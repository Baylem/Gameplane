# Agent Architecture Index

This document is the **first stop** for AI agents modifying or extending Gameplane. Instead of performing expensive repository-wide greps, use this index to find the component responsible for the feature you are working on, and read its corresponding `specs.md`.

## Core Backend Components

| Component | `specs.md` location | Primary Responsibilities |
| --- | --- | --- |
| **API Gateway** | `api/specs.md` | REST/WebSocket gateway, Auth (Local/OIDC), RBAC, DB interactions (SQLite/Postgres), Audit logging, Multi-cluster routing. **Edits to endpoints, auth, or roles go here.** |
| **Operator** | `operator/specs.md` | K8s Controller reconciling CRDs (GameServer, GameTemplate, Backup, Module, etc.). **Edits to Kubernetes resource management, CRD behavior, and K8s API integrations go here.** |
| **Agent Sidecar** | `agent/specs.md` | In-pod sidecar handling RCON, file ops, game server logs, and pod telemetry. **Edits to how the game is interacted with directly (console/files) go here.** |

## Security & Protection Components

| Component | `specs.md` location | Primary Responsibilities |
| --- | --- | --- |
| **Netguard** | `netguard/specs.md` | SSRF dial-guard preventing malicious outbound HTTP requests (e.g., stopping metadata server access during mod downloads). |
| **Gameaction** | `gameaction/specs.md` | Validates and sanitizes RCON inputs/admin commands before they hit the game server to prevent command injection. |
| **Gameproto** | `gameproto/specs.md` | Wire-protocol parsing (Minecraft/Terraria) to safely handle ping handshakes without corrupting game server connections. |

## Feature Add-ons & Sidecars

| Component | `specs.md` location | Primary Responsibilities |
| --- | --- | --- |
| **Capture Sidecar** | `capture-sidecar/specs.md` | Ephemeral network packet capture sidecar injected into game pods to capture traffic. |
| **Sentinel** | `sentinel/specs.md` | Wake-on-connect daemon holding ports for sleeping servers and waking them up upon player connection attempts. |
| **Tunnel** | `tunnel/specs.md` | Relay client supervisor for integrating with external tunnels (frp, Tailscale, playit). |
| **Audit-Syslog-Bridge** | `audit-syslog-bridge/specs.md` | Relay forwarding HTTP JSON audit events to external syslog servers. |
| **Telemetry-Receiver** | `telemetry-receiver/specs.md` | Ingests and processes anonymous usage telemetry from the API. |
| **MCP-Server** | `mcp-server/specs.md` | Read-only Model Context Protocol server exposing cluster state to AI tools. |

## Frontend & Other

| Component | `specs.md` location | Primary Responsibilities |
| --- | --- | --- |
| **Web Dashboard** | `web/specs.md` | The React SPA frontend. |
| **Modules** | `modules/<game>/specs.md` | OCI bundle templates for deploying specific games (e.g., Minecraft, Rust). |

## How to use this index
1. Identify the domain of your task (e.g., "Add an auth provider").
2. Find the component handling this domain (e.g., "API Gateway").
3. Read that component's `specs.md` (e.g., `api/specs.md`) to understand its architecture, boundaries, and specific rules before modifying code.