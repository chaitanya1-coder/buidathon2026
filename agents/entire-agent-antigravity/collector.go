package main

import (
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

// IngestSession parses Antigravity inputs through the dual-format SessionIR pipeline.
func IngestSession() (*EntireTranscript, error) {
	root, err := FindWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	session := &SessionIR{
		SessionID:   fmt.Sprintf("ag-%d", time.Now().Unix()),
		Agent:       "antigravity",
		Timestamp:   time.Now().UTC(),
		UserPrompts: []string{},
		Artifacts:   []CapturedArtifact{},
		ToolRuns:    []CapturedToolRun{},
		FilesEdited: []string{},
	}

	legacy := ingestLegacyFiles(root)
	session = MergeSessionIR(session, legacy)

	for _, streamFile := range []string{"execution.jsonl", "session.jsonl"} {
		path := filepath.Join(root, ".antigravity", streamFile)
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			session.IsPartial = true
			session.Warnings = append(session.Warnings, "failed to open "+streamFile+": "+err.Error())
			continue
		}

		streamIR, err := ParseStream(file)
		file.Close()
		if err != nil {
			session.IsPartial = true
			session.Warnings = append(session.Warnings, streamFile+" parse error: "+err.Error())
			continue
		}
		session = MergeSessionIR(session, streamIR)
	}

	if len(session.UserPrompts) == 0 &&
		len(session.Artifacts) == 0 &&
		len(session.ToolRuns) == 0 &&
		len(session.FilesEdited) == 0 {
		session.IsPartial = true
		session.Warnings = append(session.Warnings, "no ingestible session data found in .antigravity")
	}

	return session.ToEntireTranscript(), nil
}

func ingestLegacyFiles(root string) *SessionIR {
	ir := &SessionIR{
		Agent:       "antigravity",
		Timestamp:   time.Now().UTC(),
		UserPrompts: []string{},
		Artifacts:   []CapturedArtifact{},
		ToolRuns:    []CapturedToolRun{},
		FilesEdited: []string{},
	}

	antigravityDir := filepath.Join(root, ".antigravity")

	promptFile := filepath.Join(antigravityDir, "last_prompt.txt")
	if promptData, err := os.ReadFile(promptFile); err == nil {
		if prompt := strings.TrimSpace(string(promptData)); prompt != "" {
			ir.UserPrompts = append(ir.UserPrompts, prompt)
		}
	} else if !os.IsNotExist(err) {
		ir.IsPartial = true
		ir.Warnings = append(ir.Warnings, "failed to read last_prompt.txt: "+err.Error())
	}

	artifactsDir := filepath.Join(antigravityDir, "artifacts")
	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			ir.IsPartial = true
			ir.Warnings = append(ir.Warnings, "failed to read artifacts directory: "+err.Error())
		}
		return ir
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(artifactsDir, entry.Name())
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			ir.IsPartial = true
			ir.Warnings = append(ir.Warnings, "failed to read artifact "+entry.Name()+": "+readErr.Error())
			continue
		}
		ir.Artifacts = append(ir.Artifacts, CapturedArtifact{
			Name:    entry.Name(),
			Type:    artifactTypeFromName(entry.Name()),
			Content: string(content),
		})
	}

	return ir
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
