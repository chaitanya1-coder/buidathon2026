package main

import (
	"bytes"
	"os"
	"path/filepath"
)

// FindWorkspaceRoot ascends directories to find .antigravity or .git
func FindWorkspaceRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".antigravity")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return os.Getwd()
}

func IngestSession() (*SessionIR, error) {
	root, err := FindWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	agDir := filepath.Join(root, ".antigravity")

	// Case 1: Check for V2 Unified Event Stream (`events.jsonl` or `stream.jsonl`)
	for _, streamName := range []string{"events.jsonl", "stream.jsonl"} {
		v2StreamPath := filepath.Join(agDir, streamName)
		if streamData, err := os.ReadFile(v2StreamPath); err == nil {
			return ParseStream(bytes.NewReader(streamData))
		}
	}

	// Case 2: Fallback to V1 (Legacy files: execution.jsonl + artifacts/ + last_prompt.txt)
	ir := &SessionIR{
		SessionID:   "ag-v1-legacy",
		Agent:       "antigravity",
		UserPrompts: []string{},
		Artifacts:   []CapturedArtifact{},
		ToolRuns:    []CapturedToolRun{},
	}

	// Read legacy prompt
	if pData, err := os.ReadFile(filepath.Join(agDir, "last_prompt.txt")); err == nil {
		ir.UserPrompts = append(ir.UserPrompts, string(pData))
	}

	// Read legacy artifacts
	artDir := filepath.Join(agDir, "artifacts")
	if entries, err := os.ReadDir(artDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, _ := os.ReadFile(filepath.Join(artDir, entry.Name()))
			ir.Artifacts = append(ir.Artifacts, CapturedArtifact{
				Name:    entry.Name(),
				Type:    "legacy_artifact",
				Content: string(content),
			})
		}
	}

	// Read legacy execution log
	if logData, err := os.ReadFile(filepath.Join(agDir, "execution.jsonl")); err == nil {
		parsed, _ := ParseStream(bytes.NewReader(logData))
		ir.ToolRuns = parsed.ToolRuns
	}

	return ir, nil
}
