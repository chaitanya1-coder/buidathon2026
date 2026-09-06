package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// IngestSession parses Antigravity artifacts and session telemetry
func IngestSession() (*EntireTranscript, error) {
	root, err := FindWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	antigravityDir := filepath.Join(root, ".antigravity")
	transcript := &EntireTranscript{
		SessionID:   fmt.Sprintf("ag-%d", time.Now().Unix()),
		Agent:       "antigravity",
		Timestamp:   time.Now().UTC(),
		UserPrompts: []string{},
		Artifacts:   []CapturedArtifact{},
		ToolRuns:    []CapturedToolRun{},
		FilesEdited: []string{},
	}

	// 1. Ingest Prompt / Intent if saved
	promptFile := filepath.Join(antigravityDir, "last_prompt.txt")
	if promptData, err := os.ReadFile(promptFile); err == nil {
		transcript.UserPrompts = append(transcript.UserPrompts, strings.TrimSpace(string(promptData)))
	}

	// 2. Ingest Structured Artifacts (Task Lists, Plans)
	artifactsDir := filepath.Join(antigravityDir, "artifacts")
	if entries, err := os.ReadDir(artifactsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(artifactsDir, entry.Name())
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				continue
			}

			artType := "plan"
			if strings.Contains(entry.Name(), "task") {
				artType = "task"
			} else if strings.Contains(entry.Name(), "verify") {
				artType = "verification"
			}

			transcript.Artifacts = append(transcript.Artifacts, CapturedArtifact{
				Name:    entry.Name(),
				Type:    artType,
				Content: string(content),
			})
		}
	}

	// 3. Ingest Execution & Tool Logs (Negative knowledge & bash execution)
	logFile := filepath.Join(antigravityDir, "execution.jsonl")
	if f, err := os.Open(logFile); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var record struct {
				Tool     string `json:"tool"`
				Command  string `json:"command"`
				ExitCode int    `json:"exit_code"`
				Output   string `json:"output"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &record); err == nil {
				transcript.ToolRuns = append(transcript.ToolRuns, CapturedToolRun{
					ToolName: record.Tool,
					Command:  record.Command,
					ExitCode: record.ExitCode,
					Output:   record.Output,
				})
			}
		}
	}

	return transcript, nil
}
