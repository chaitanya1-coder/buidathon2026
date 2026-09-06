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

type SessionTranscript struct {
	SessionID string           `json:"session_id"`
	Steps     []TranscriptStep `json:"steps"`
}

func findTranscriptFile(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}

	// Direct file path if sessionID ends with transcript.jsonl
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

	// Also check relative or absolute directory path
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
	filePath, err := findTranscriptFile(sessionID)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open transcript file %s: %w", filePath, err)
	}
	defer file.Close()

	var steps []TranscriptStep
	scanner := bufio.NewScanner(file)
	// Set larger buffer size in case of long content lines
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var step TranscriptStep
		if err := json.Unmarshal(line, &step); err != nil {
			// Skip unparseable lines or handle gracefully
			continue
		}
		steps = append(steps, step)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading transcript file %s: %w", filePath, err)
	}

	result := SessionTranscript{
		SessionID: sessionID,
		Steps:     steps,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
