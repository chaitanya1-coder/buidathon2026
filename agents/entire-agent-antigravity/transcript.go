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

type EntireSessionTranscript struct {
	SessionID   string        `json:"session_id"`
	Agent       string        `json:"agent"`
	Timestamp   time.Time     `json:"timestamp"`
	Prompts     []string      `json:"prompts"`
	Artifacts   []ArtifactDoc `json:"artifacts"`
	ToolRuns    []ToolRecord  `json:"tool_runs"`
	FilesEdited []string      `json:"files_edited"`
}

type ArtifactDoc struct {
	Type    string `json:"type"` // "task_list", "implementation_plan", "verification"
	Content string `json:"content"`
}

type ToolRecord struct {
	ToolName string `json:"tool_name"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

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

func parseAntigravityExecutionLogs(logData []byte) []ToolRecord {
	var toolRuns []ToolRecord
	lines := strings.Split(string(logData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record ToolRecord
		if err := json.Unmarshal([]byte(line), &record); err == nil && record.ToolName != "" {
			toolRuns = append(toolRuns, record)
			continue
		}
		// Plain text log format fallback parsing: [TOOL] ToolName | Input | Output | ExitCode
		if strings.HasPrefix(line, "[TOOL]") {
			parts := strings.SplitN(line, "|", 4)
			rec := ToolRecord{}
			if len(parts) > 0 {
				rec.ToolName = strings.TrimSpace(strings.TrimPrefix(parts[0], "[TOOL]"))
			}
			if len(parts) > 1 {
				rec.Input = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				rec.Output = strings.TrimSpace(parts[2])
			}
			toolRuns = append(toolRuns, rec)
		}
	}
	return toolRuns
}

func CollectAntigravitySession(workspaceRoot string) (*EntireSessionTranscript, error) {
	transcript := &EntireSessionTranscript{
		Agent:     "antigravity",
		Timestamp: time.Now(),
	}

	// 1. Ingest Task List and Implementation Plan Artifacts
	artifactsDir := filepath.Join(workspaceRoot, ".antigravity", "artifacts")
	if files, err := os.ReadDir(artifactsDir); err == nil {
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(artifactsDir, file.Name()))
			if err == nil {
				transcript.Artifacts = append(transcript.Artifacts, ArtifactDoc{
					Type:    file.Name(),
					Content: string(content),
				})
			}
		}
	}

	// 2. Ingest Terminal / Execution logs generated during agent run
	logPath := filepath.Join(workspaceRoot, ".antigravity", "execution.log")
	if logData, err := os.ReadFile(logPath); err == nil {
		// Parse tool events, exit codes, and output blocks
		transcript.ToolRuns = parseAntigravityExecutionLogs(logData)
	}

	return transcript, nil
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
	workspaceRoot, _ := os.Getwd()
	sessionTranscript, err := CollectAntigravitySession(workspaceRoot)
	if err != nil {
		return fmt.Errorf("failed collecting session: %w", err)
	}
	sessionTranscript.SessionID = sessionID

	// Try reading detailed transcript steps and session artifacts from session directory if available
	filePath, err := findTranscriptFile(sessionID)
	if err == nil {
		sessionDir := filepath.Dir(filepath.Dir(filepath.Dir(filePath)))
		if files, err := os.ReadDir(sessionDir); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
					if content, err := os.ReadFile(filepath.Join(sessionDir, f.Name())); err == nil {
						sessionTranscript.Artifacts = append(sessionTranscript.Artifacts, ArtifactDoc{
							Type:    f.Name(),
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
						sessionTranscript.Prompts = append(sessionTranscript.Prompts, step.Content)
					}
					for _, tc := range step.ToolCalls {
						toolInput, _ := json.Marshal(tc.Args)
						sessionTranscript.ToolRuns = append(sessionTranscript.ToolRuns, ToolRecord{
							ToolName: tc.Name,
							Input:    string(toolInput),
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
