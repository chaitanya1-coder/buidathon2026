package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// SessionIR is the unified representation consumed by Checkpoint writers
type SessionIR struct {
	SessionID   string             `json:"session_id"`
	Agent       string             `json:"agent"`
	Timestamp   time.Time          `json:"timestamp"`
	UserPrompts []string           `json:"user_prompts"`
	Artifacts   []CapturedArtifact `json:"artifacts"`
	ToolRuns    []CapturedToolRun  `json:"tool_runs"`
	FilesEdited []string           `json:"files_edited"`
	IsPartial   bool               `json:"is_partial,omitempty"`
	Warnings    []string           `json:"warnings,omitempty"`
}

// FormatVersion detector
type FormatVersion string

const (
	FormatV1Legacy FormatVersion = "v1"
	FormatV2Stream FormatVersion = "v2"
)

// V2Event represents the envelope for new lifecycle events
type V2LifecycleEvent struct {
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"` // e.g. "session_init", "prompt", "artifact_patch", "tool_result"
	SessionID string                 `json:"session_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

// ParseStream reads lines defensively, handling both formats, unknown events, and truncated files
func ParseStream(r io.Reader) (*SessionIR, error) {
	ir := &SessionIR{
		Agent:       "antigravity",
		Timestamp:   time.Now().UTC(),
		UserPrompts: make([]string, 0),
		Artifacts:   make([]CapturedArtifact, 0),
		ToolRuns:    make([]CapturedToolRun, 0),
		FilesEdited: make([]string, 0),
	}

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse envelope
		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawMap); err != nil {
			// Incomplete / truncated line: record warning and continue to produce partial result
			ir.IsPartial = true
			ir.Warnings = append(ir.Warnings, "truncated or corrupted record skipped at line")
			continue
		}

		// Detect format: V2 has a top-level "type" with a nested "payload", V1 has "tool" or file properties
		if eventType, hasType := rawMap["type"].(string); hasType {
			processV2Event(eventType, rawMap, ir)
		} else if _, hasTool := rawMap["tool"]; hasTool {
			processV1ToolEvent(rawMap, ir)
		} else {
			// Unknown event format: MUST NOT CRASH
			ir.Warnings = append(ir.Warnings, "unrecognized event signature ignored")
		}
	}

	if err := scanner.Err(); err != nil {
		// Stream ended unexpectedly (incomplete input) -> preserve whatever was collected
		ir.IsPartial = true
		ir.Warnings = append(ir.Warnings, "stream ended prematurely: "+err.Error())
	}

	if ir.SessionID == "" {
		ir.SessionID = "ag-session-recovered"
	}

	return ir, nil
}

func processV2Event(eventType string, raw map[string]interface{}, ir *SessionIR) {
	// Safely extract payload without panicking
	payload, _ := raw["payload"].(map[string]interface{})
	if payload == nil {
		payload = raw // fallback if payload is top-level
	}

	switch eventType {
	case "session.start", "session_init":
		if id, ok := payload["session_id"].(string); ok {
			ir.SessionID = id
		}
	case "prompt.received", "user_prompt", "prompt":
		prompt := stringField(payload, "prompt", "text", "content")
		if prompt != "" {
			ir.UserPrompts = append(ir.UserPrompts, prompt)
		}
	case "artifact.update", "artifact_emitted", "artifact_update", "artifact":
		name := stringField(payload, "name", "artifact_name")
		artType := stringField(payload, "artifact_type", "type_hint")
		content := stringField(payload, "content", "body")
		if artType == "" {
			artType = artifactTypeFromName(name)
		}
		if name != "" {
			ir.Artifacts = append(ir.Artifacts, CapturedArtifact{
				Name:    name,
				Type:    artType,
				Content: content,
			})
		}
	case "tool.execution", "tool_result", "tool_call", "tool_run":
		toolName := stringField(payload, "tool", "tool_name")
		cmd := stringField(payload, "command", "input")
		out := stringField(payload, "output", "result")
		exitCode := intField(payload["exit_code"])
		ir.ToolRuns = append(ir.ToolRuns, CapturedToolRun{
			ToolName: toolName,
			Command:  cmd,
			ExitCode: exitCode,
			Output:   out,
		})
	case "file_edit", "file_changed":
		if path := stringField(payload, "path", "file", "target_file"); path != "" {
			ir.FilesEdited = append(ir.FilesEdited, path)
		}
	default:
		// UNKNOWN EVENTS: Must not crash, quietly absorb or record warning
		ir.Warnings = append(ir.Warnings, "unhandled v2 event type: "+eventType)
	}
}

func processV1ToolEvent(raw map[string]interface{}, ir *SessionIR) {
	tool, _ := raw["tool"].(string)
	cmd, _ := raw["command"].(string)
	out, _ := raw["output"].(string)
	exitCode := intField(raw["exit_code"])
	ir.ToolRuns = append(ir.ToolRuns, CapturedToolRun{
		ToolName: tool,
		Command:  cmd,
		ExitCode: exitCode,
		Output:   out,
	})
}

// MergeSessionIR combines stream and legacy directory ingestion results.
func MergeSessionIR(base, overlay *SessionIR) *SessionIR {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	if overlay.SessionID != "" && overlay.SessionID != "ag-session-recovered" {
		base.SessionID = overlay.SessionID
	}
	if overlay.Agent != "" {
		base.Agent = overlay.Agent
	}
	if !overlay.Timestamp.IsZero() {
		base.Timestamp = overlay.Timestamp
	}
	if overlay.IsPartial {
		base.IsPartial = true
	}

	base.UserPrompts = appendUniqueStrings(base.UserPrompts, overlay.UserPrompts)
	base.Artifacts = appendUniqueCapturedArtifacts(base.Artifacts, overlay.Artifacts)
	base.ToolRuns = appendUniqueCapturedToolRuns(base.ToolRuns, overlay.ToolRuns)
	base.FilesEdited = appendUniqueStrings(base.FilesEdited, overlay.FilesEdited)
	base.Warnings = appendUniqueStrings(base.Warnings, overlay.Warnings)

	return base
}

// ToEntireTranscript projects SessionIR into the Entire checkpoint schema.
func (ir *SessionIR) ToEntireTranscript() *EntireTranscript {
	if ir == nil {
		return &EntireTranscript{
			Agent:       "antigravity",
			Timestamp:   time.Now().UTC(),
			UserPrompts: []string{},
			Artifacts:   []CapturedArtifact{},
			ToolRuns:    []CapturedToolRun{},
			FilesEdited: []string{},
		}
	}

	return &EntireTranscript{
		SessionID:   ir.SessionID,
		Agent:       ir.Agent,
		Timestamp:   ir.Timestamp,
		UserPrompts: append([]string{}, ir.UserPrompts...),
		Artifacts:   append([]CapturedArtifact{}, ir.Artifacts...),
		ToolRuns:    append([]CapturedToolRun{}, ir.ToolRuns...),
		FilesEdited: append([]string{}, ir.FilesEdited...),
	}
}

func stringField(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func intField(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func appendUniqueStrings(dst, src []string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[item] = struct{}{}
	}
	for _, item := range src {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func appendUniqueCapturedArtifacts(dst, src []CapturedArtifact) []CapturedArtifact {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[capturedArtifactKey(item)] = struct{}{}
	}
	for _, item := range src {
		key := capturedArtifactKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func appendUniqueCapturedToolRuns(dst, src []CapturedToolRun) []CapturedToolRun {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[capturedToolRunKey(item)] = struct{}{}
	}
	for _, item := range src {
		key := capturedToolRunKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func artifactTypeFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "task"):
		return "task"
	case strings.Contains(lower, "plan"):
		return "plan"
	case strings.Contains(lower, "verify"):
		return "verification"
	default:
		return "artifact"
	}
}

func capturedArtifactKey(artifact CapturedArtifact) string {
	return artifact.Name + "|" + artifact.Type + "|" + artifact.Content
}

func capturedToolRunKey(run CapturedToolRun) string {
	return run.ToolName + "|" + run.Command + "|" + run.Output + "|" + fmt.Sprintf("%d", run.ExitCode)
}
