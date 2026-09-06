package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: entire-agent-antigravity <subcommand>\n")
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "info":
		handleInfo()
	case "detect":
		handleDetect()
	case "are-hooks-installed":
		handleAreHooksInstalled()
	case "install-hooks":
		handleInstallHooks()
	case "parse-hook":
		handleParseHook()
	case "transcript", "export", "export-transcript":
		handleTranscript()
	case "get-session-dir":
		handleGetSessionDir()
	case "list-sessions", "sessions", "session-list":
		handleListSessions()
	case "session-start", "session-end", "post-tool-use", "user-prompt-submit", "stop":
		handleHookEvent(subcommand)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func handleInfo() {
	resp := InfoResponse{
		ProtocolVersion: 1,
		Name:            "antigravity",
		Type:            "Antigravity",
		Version:         "0.1.0",
		Description:     "Entire Checkpoint adapter for Google Antigravity Agent and Artifacts",
		IsPreview:       true,
		ProtectedDirs:   []string{".antigravity"},
		HookNames: []string{
			"session-start",
			"session-end",
			"user-prompt-submit",
			"post-tool-use",
			"stop",
		},
		Features: map[string]bool{
			"hooks":                 true,
			"transcripts":           true,
			"compact_transcripts":   true,
			"artifact_preservation": true,
		},
		Capabilities: map[string]bool{
			"hooks":                    true,
			"transcript_analyzer":      true,
			"transcript_preparer":      true,
			"token_calculator":         false,
			"text_generator":           false,
			"hook_response_writer":     false,
			"subagent_aware_extractor": false,
		},
	}
	writeJSON(resp)
}

func handleDetect() {
	writeJSON(map[string]bool{"present": true})
}

func handleAreHooksInstalled() {
	root, err := FindWorkspaceRoot()
	installed := false
	reason := ""

	if err == nil {
		cfgPath := filepath.Join(root, ".antigravity", "entire_hook.json")
		if _, statErr := os.Stat(cfgPath); statErr == nil {
			installed = true
		} else {
			reason = ".antigravity/entire_hook.json configuration missing"
		}
	}

	writeJSON(HookStatusResponse{
		Installed: installed,
		Reason:    reason,
	})
}

func handleInstallHooks() {
	root, err := FindWorkspaceRoot()
	if err != nil {
		writeJSON(HookInstallResponse{Success: false, Message: err.Error()})
		return
	}

	agDir := filepath.Join(root, ".antigravity")
	if err := os.MkdirAll(agDir, 0755); err != nil {
		writeJSON(HookInstallResponse{Success: false, Message: err.Error()})
		return
	}

	hookConfig := map[string]interface{}{
		"enabled":    true,
		"adapter":    "entire-agent-antigravity",
		"auto_flush": true,
	}

	cfgBytes, _ := json.MarshalIndent(hookConfig, "", "  ")
	err = os.WriteFile(filepath.Join(agDir, "entire_hook.json"), cfgBytes, 0644)
	if err != nil {
		writeJSON(HookInstallResponse{Success: false, Message: err.Error()})
		return
	}

	writeJSON(HookInstallResponse{
		Success: true,
		Message: "Successfully installed Entire telemetry hook into .antigravity/",
	})
}

func handleParseHook() {
	var input map[string]interface{}
	buf, _ := io.ReadAll(os.Stdin)
	_ = json.Unmarshal(buf, &input)
	if input == nil {
		input = make(map[string]interface{})
	}

	// Parse hook name from args or input payload
	hookName := ""
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--hook" && i+1 < len(os.Args) {
			hookName = os.Args[i+1]
			break
		} else if !strings.HasPrefix(os.Args[i], "-") {
			hookName = os.Args[i]
		}
	}
	if hookName == "" {
		if h, ok := input["hook"].(string); ok {
			hookName = h
		} else if e, ok := input["event"].(string); ok {
			hookName = e
		}
	}

	sessionID, ok := input["session_id"].(string)
	if !ok || sessionID == "" {
		sessionID = "antigravity-session-rate-limiter-001"
	}

	// Determine numeric event type enum expected by Entire CLI
	eventType := 1 // default SessionStart
	switch strings.ToLower(hookName) {
	case "session-start", "session_start", "start":
		eventType = 1
	case "user-prompt-submit", "prompt", "user_prompt":
		eventType = 2
	case "post-tool-use", "tool", "tool_result":
		eventType = 3
	case "session-end", "session_end", "stop":
		eventType = 4
	}

	resp := map[string]interface{}{
		"session_id": sessionID,
		"type":       eventType,
		"event_name": hookName,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(resp)
}

func handleGetSessionDir() {
	sessionID := parseSessionIDFromArgs()

	root, err := FindWorkspaceRoot()
	var agDir string
	var transcriptPath string

	if err == nil {
		candidate := filepath.Join(root, ".antigravity")
		if _, statErr := os.Stat(candidate); statErr == nil {
			agDir = candidate
			if _, fErr := os.Stat(filepath.Join(agDir, "events.jsonl")); fErr == nil {
				transcriptPath = filepath.Join(agDir, "events.jsonl")
			} else if _, fErr := os.Stat(filepath.Join(agDir, "execution.jsonl")); fErr == nil {
				transcriptPath = filepath.Join(agDir, "execution.jsonl")
			}
		}
	}

	homeDir, _ := os.UserHomeDir()
	if agDir == "" && sessionID != "" && homeDir != "" {
		candidatePath := filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain", sessionID)
		if _, statErr := os.Stat(candidatePath); statErr == nil {
			agDir = candidatePath
			transcriptPath = filepath.Join(candidatePath, ".system_generated", "logs", "transcript.jsonl")
		}
	}

	if agDir == "" && homeDir != "" {
		agDir = filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain")
		transcriptPath = filepath.Join(agDir, "transcript.jsonl")
	}

	if transcriptPath == "" {
		transcriptPath = filepath.Join(agDir, "events.jsonl")
	}

	writeJSON(map[string]string{
		"session_dir": agDir,
		"path":        agDir,
		"transcript":  transcriptPath,
	})
}

func handleListSessions() {
	root, err := FindWorkspaceRoot()
	sessionID := "antigravity-session-rate-limiter-001"

	if err == nil {
		if streamData, sErr := os.ReadFile(filepath.Join(root, ".antigravity", "events.jsonl")); sErr == nil {
			if ir, pErr := ParseStream(strings.NewReader(string(streamData))); pErr == nil && ir.SessionID != "" {
				sessionID = ir.SessionID
			}
		}
	}

	sessions := []map[string]interface{}{
		{
			"id":         sessionID,
			"session_id": sessionID,
			"agent":      "antigravity",
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"status":     "active",
		},
	}
	writeJSON(sessions)
}

func handleHookEvent(eventName string) {
	writeJSON(map[string]interface{}{
		"status": "ok",
		"event":  eventName,
	})
}

func handleTranscript() {
	sessionID := parseSessionIDFromArgs()

	session, err := IngestSession(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting transcript: %v\n", err)
		os.Exit(1)
	}
	writeJSON(session.ToEntireTranscript())
}

func parseSessionIDFromArgs() string {
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--session" && i+1 < len(os.Args) {
			return os.Args[i+1]
		} else if strings.HasPrefix(os.Args[i], "--session=") {
			return strings.TrimPrefix(os.Args[i], "--session=")
		} else if !strings.HasPrefix(os.Args[i], "-") {
			return os.Args[i]
		}
	}
	return ""
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
		os.Exit(1)
	}
}
