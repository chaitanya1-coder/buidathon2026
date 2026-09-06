package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func IngestSession(sessionID string) (*SessionIR, error) {
	homeDir, _ := os.UserHomeDir()

	// Case 1: Session ID provided, check IDE brain transcript log
	if sessionID != "" && homeDir != "" {
		brainLogPath := filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
		if data, err := os.ReadFile(brainLogPath); err == nil {
			ir, err := ParseStream(bytes.NewReader(data))
			if err == nil {
				ir.SessionID = sessionID
				return ir, nil
			}
		}
	}

	root, err := FindWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	agDir := filepath.Join(root, ".antigravity")

	// Case 2: Check for V2 Unified Event Stream (`events.jsonl` or `stream.jsonl` or `<session_id>.jsonl`)
	var candidateStreams []string
	if sessionID != "" {
		candidateStreams = append(candidateStreams, sessionID+".jsonl", sessionID)
	}
	candidateStreams = append(candidateStreams, "events.jsonl", "stream.jsonl")

	for _, streamName := range candidateStreams {
		v2StreamPath := filepath.Join(agDir, streamName)
		if streamData, err := os.ReadFile(v2StreamPath); err == nil {
			ir, err := ParseStream(bytes.NewReader(streamData))
			if err == nil {
				if sessionID != "" {
					ir.SessionID = sessionID
				}
				return ir, nil
			}
		}
	}

	// Case 3: Fallback to V1 (Legacy files: execution.jsonl + artifacts/ + last_prompt.txt)
	ir := &SessionIR{
		SessionID:   sessionID,
		Agent:       "antigravity",
		UserPrompts: []string{},
		Artifacts:   []CapturedArtifact{},
		ToolRuns:    []CapturedToolRun{},
	}
	if ir.SessionID == "" {
		ir.SessionID = "antigravity-session-rate-limiter-001"
	}

	// Read legacy prompt
	if pData, err := os.ReadFile(filepath.Join(agDir, "last_prompt.txt")); err == nil {
		ir.UserPrompts = append(ir.UserPrompts, strings.TrimSpace(string(pData)))
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
				Type:    artifactTypeFromName(entry.Name()),
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
