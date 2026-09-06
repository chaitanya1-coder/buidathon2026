package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// V1Parser ingests legacy directory artifacts and execution.jsonl logs.
type V1Parser struct{}

func (p *V1Parser) Parse(root string) (*SessionIR, error) {
	session := NewSessionIR()
	antigravityDir := filepath.Join(root, ".antigravity")

	promptFile := filepath.Join(antigravityDir, "last_prompt.txt")
	if promptData, err := os.ReadFile(promptFile); err == nil {
		if prompt := strings.TrimSpace(string(promptData)); prompt != "" {
			session.UserPrompts = append(session.UserPrompts, prompt)
		}
	} else if !os.IsNotExist(err) {
		session.MarkPartial("failed to read last_prompt.txt: " + err.Error())
	}

	artifactsDir := filepath.Join(antigravityDir, "artifacts")
	artifactEntries, err := os.ReadDir(artifactsDir)
	if err != nil {
		if os.IsNotExist(err) {
			session.MarkPartial("legacy artifacts directory missing")
		} else {
			session.MarkPartial("failed to read artifacts directory: " + err.Error())
		}
	} else {
		for _, entry := range artifactEntries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(artifactsDir, entry.Name())
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				session.MarkPartial("failed to read artifact " + entry.Name() + ": " + readErr.Error())
				continue
			}
			session.Artifacts = append(session.Artifacts, IRArtifact{
				Name:    entry.Name(),
				Type:    artifactTypeFromName(entry.Name()),
				Content: string(content),
			})
		}
	}

	logFile := filepath.Join(antigravityDir, "execution.jsonl")
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			session.MarkPartial("legacy execution.jsonl missing")
		} else {
			session.MarkPartial("failed to open execution.jsonl: " + err.Error())
		}
		return session, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	parsedLines := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record struct {
			Tool     string `json:"tool"`
			Command  string `json:"command"`
			ExitCode int    `json:"exit_code"`
			Output   string `json:"output"`
		}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			session.MarkPartial("skipped malformed execution.jsonl line")
			session.RawEvents = append(session.RawEvents, RawEvent{
				Type:    "legacy_execution_line",
				RawLine: line,
				Payload: map[string]interface{}{"parse_error": err.Error()},
			})
			continue
		}

		parsedLines++
		session.ToolRuns = append(session.ToolRuns, IRToolRun{
			ToolName: firstNonEmpty(record.Tool, "unknown"),
			Command:  record.Command,
			ExitCode: record.ExitCode,
			Output:   record.Output,
		})
	}
	if err := scanner.Err(); err != nil {
		session.MarkPartial("execution.jsonl scan error: " + err.Error())
	}
	if parsedLines == 0 {
		session.MarkPartial("execution.jsonl contained no parseable tool runs")
	}

	return session, nil
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
