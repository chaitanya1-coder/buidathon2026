package main

import "time"

// Protocol response schemas
type InfoResponse struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Features    map[string]bool `json:"features"`
}

type HookStatusResponse struct {
	Installed bool   `json:"installed"`
	Reason    string `json:"reason,omitempty"`
}

type HookInstallResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Entire Checkpoint Transcript schema
type EntireTranscript struct {
	SessionID   string             `json:"session_id"`
	Agent       string             `json:"agent"`
	Timestamp   time.Time          `json:"timestamp"`
	UserPrompts []string           `json:"user_prompts"`
	Artifacts   []CapturedArtifact `json:"artifacts"`
	ToolRuns    []CapturedToolRun  `json:"tool_runs"`
	FilesEdited []string           `json:"files_edited"`
}

type CapturedArtifact struct {
	Name    string `json:"name"`    // e.g. "task_list.md", "implementation_plan.md"
	Type    string `json:"type"`    // "plan" | "task" | "verification"
	Content string `json:"content"` // Raw markdown content
}

type CapturedToolRun struct {
	ToolName string `json:"tool_name"`
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}
