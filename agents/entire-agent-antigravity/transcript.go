package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type TranscriptStep struct {
	StepIndex int        `json:"step_index"`
	Source    string     `json:"source"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func artifactTypeFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "task"):
		return "task"
	case strings.Contains(lower, "plan"):
		return "plan"
	case strings.Contains(lower, "verification"):
		return "verification"
	default:
		return "artifact"
	}
}

func findTranscriptFile(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}

	if strings.HasSuffix(sessionID, "transcript.jsonl") {
		if _, err := os.Stat(sessionID); err == nil {
			return sessionID, nil
		}
	}

	homeDir, _ := os.UserHomeDir()
	appDataDir := os.Getenv("ANTIGRAVITY_APP_DATA_DIR")

	var candidatePaths []string

	if appDataDir != "" {
		candidatePaths = append(candidatePaths,
			filepath.Join(appDataDir, "brain", sessionID, ".system_generated", "logs", "transcript.jsonl"),
			filepath.Join(appDataDir, "brain", sessionID, "transcript.jsonl"),
		)
	}

	if homeDir != "" {
		candidatePaths = append(candidatePaths,
			filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl"),
			filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain", sessionID, "transcript.jsonl"),
		)
	}

	candidatePaths = append(candidatePaths,
		filepath.Join(sessionID, ".system_generated", "logs", "transcript.jsonl"),
		filepath.Join(sessionID, "transcript.jsonl"),
	)

	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("transcript file for session %s not found in standard locations", sessionID)
}

func handleTranscript(sessionID string) error {
	sessionTranscript, err := IngestSession()
	if err != nil {
		return fmt.Errorf("failed collecting session: %w", err)
	}
	if sessionID != "" {
		sessionTranscript.SessionID = sessionID
	}

	// Try reading detailed transcript steps and session artifacts from session directory if available
	filePath, err := findTranscriptFile(sessionID)
	if err == nil {
		sessionDir := filepath.Dir(filepath.Dir(filepath.Dir(filePath)))
		if files, err := os.ReadDir(sessionDir); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
					if content, err := os.ReadFile(filepath.Join(sessionDir, f.Name())); err == nil {
						sessionTranscript.Artifacts = append(sessionTranscript.Artifacts, CapturedArtifact{
							Name:    f.Name(),
							Type:    artifactTypeFromName(f.Name()),
							Content: string(content),
						})
					}
				}
			}
		}

		if file, err := os.Open(filePath); err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			const maxCapacity = 10 * 1024 * 1024
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, maxCapacity)

			editedFilesMap := make(map[string]bool)

			for scanner.Scan() {
				line := scanner.Bytes()
				if len(strings.TrimSpace(string(line))) == 0 {
					continue
				}

				var step TranscriptStep
				if err := json.Unmarshal(line, &step); err == nil {
					if step.Type == "USER_INPUT" && step.Content != "" {
						sessionTranscript.UserPrompts = append(sessionTranscript.UserPrompts, step.Content)
					}
					for _, tc := range step.ToolCalls {
						toolCommand, _ := json.Marshal(tc.Args)
						sessionTranscript.ToolRuns = append(sessionTranscript.ToolRuns, CapturedToolRun{
							ToolName: tc.Name,
							Command:  string(toolCommand),
							Output:   step.Content,
							ExitCode: 0,
						})
						if targetFile, ok := tc.Args["TargetFile"].(string); ok {
							editedFilesMap[targetFile] = true
						}
					}
				}
			}

			for f := range editedFilesMap {
				sessionTranscript.FilesEdited = append(sessionTranscript.FilesEdited, f)
			}
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sessionTranscript)
}
