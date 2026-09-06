# entire-agent-antigravity

> **Entire Hackathon Submission — Track 3: Bring Entire to a New Agent or Workflow**  
> Checkpoint-native adapter bridging Google Antigravity Agent sessions, structured artifacts, and execution telemetry into Entire Checkpoints.

---

## Overview

AI-native IDEs and agents like **Google Antigravity** produce rich contextual assets during development: task checklists, architectural implementation plans, and terminal execution trails. However, traditional version control treats this metadata as ephemeral—standard `git commit` workflows discard why code changed, which architectural dead-ends were abandoned, and whether tests failed during intermediate steps.

`entire-agent-antigravity` is a native adapter built for the Entire ecosystem (`entireio/external-agents`). It captures in-flight Antigravity artifacts and execution traces, persisting them directly into Git metadata under `refs/entire/checkpoints/*`.

---

## Key Capabilities

* **Full Artifact Preservation:** Automatically discovers and embeds Antigravity `task_list.md`, `implementation_plan.md`, and verification artifacts into checkpoints.
* **Negative Knowledge Capture:** Logs failed terminal commands, stack traces, and non-zero exit codes from agent execution loops so developers and downstream agents avoid repeating discarded approaches.
* **Dual-Format Ingestion Engine:** Supports legacy discrete file structures (V1) and unified lifecycle JSONL event streams (V2) through a shared Intermediate Representation (IR).
* **Fault-Tolerant Parsing:**
  * **Unknown Event Resiliency:** Silently skips unrecognized schema versions and telemetry frames without panicking.
  * **Truncated Stream Recovery:** Yields usable partial checkpoints if a process terminates abruptly mid-session rather than discarding the context.
* **Entire Protocol Native:** Fully implements `info`, `are-hooks-installed`, `install-hooks`, and `transcript` CLI specifications via standard I/O.

---

## Getting Started

### Prerequisites

* Go 1.21+
* Git 2.30+
* Entire CLI (`entireio/cli`) installed and available on `$PATH`

### 1. Build and Install the Binary

Entire expects external agent binaries on `$PATH` following the naming format `entire-agent-<name>`.

```bash
cd agents/entire-agent-antigravity
go build -o entire-agent-antigravity .

# Move to your system PATH (e.g., GOPATH or /usr/local/bin)
export PATH="$PWD:$PATH"
cp entire-agent-antigravity $(go env GOPATH)/bin/
Verify that the CLI protocol responds:Bashentire-agent-antigravity info
Expected output:JSON{
  "name": "antigravity",
  "version": "0.1.0",
  "description": "Entire Checkpoint adapter for Google Antigravity Agent and Artifacts",
  "features": {
    "artifact_preservation": true,
    "compact_transcripts": true,
    "hooks": true,
    "transcripts": true
  }
}
2. Enable in Your RepositoryIn any project managed by Entire:Ensure external agent discovery is enabled in .entire/settings.json:JSON{
  "external_agents": true
}
Register and configure the hook:Bashentire enable --agent antigravity
Usage & VerificationIngesting In-Flight Development SessionsWhen an Antigravity agent edits a repository, it generates plan artifacts and tool logs. When committing or running Entire commands:Bash# Verify hook status
entire-agent-antigravity are-hooks-installed

# Inspect generated checkpoint transcript manually
entire-agent-antigravity transcript
To create an Entire Checkpoint using Git:Bashgit add .
git commit -m "feat(auth): implement token bucket rate limiter"
Inspect the attached metadata:Bashentire checkpoint list
entire checkpoint show HEAD --json
Testing & ReliabilityThe parser includes unit tests verifying zero-crash guarantees against ambiguous agent logs:V1 Legacy Format: Validates parsing of split artifact directories and execution.jsonl.V2 Stream Format: Validates parsing of event streams with envelope dispatching.Unknown Event Handling: Verifies that unmapped schema updates do not crash the binary.Incomplete Input Handling: Ensures streams truncated mid-token produce valid partial transcripts.Run the test suite:Bashgo test -v ./...
To run against a raw JSONL fixture:Bashcat fixture.jsonl | entire-agent-antigravity transcript
Track Compliance Checklist (Track 3)RequirementImplementation DetailStatusNew Agent IntegrationIntegrates Google Antigravity IDE and CLI runtime into EntirePassEntire Fork NativeImplemented as an idiomatic binary inside entireio/external-agentsPassContext Beyond DiffPreserves Antigravity task states, architecture plans, and negative tool outputsPassDual Format CompatibilityHandles both legacy discrete files and modern JSONL streamsPassResilience & FallbacksZero-panic guarantee on unknown events; produces partial snapshots on truncated logsPass

