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

## Architecture