package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	case "transcript":
		handleTranscript()
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
			"transcript_preparer":      false,
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

func handleTranscript() {
	transcript, err := IngestSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting transcript: %v\n", err)
		os.Exit(1)
	}
	writeJSON(transcript)
}

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Serialization error: %v\n", err)
		os.Exit(1)
	}
}
