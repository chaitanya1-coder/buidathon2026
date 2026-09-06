package main

import (
	"fmt"
	"time"
)

// SessionIR is the canonical intermediate representation for Antigravity sessions.
// Both V1 (legacy files) and V2 (unified JSONL) parsers populate this structure.
type SessionIR struct {
	SessionID   string
	Agent       string
	Timestamp   time.Time
	Partial     bool
	UserPrompts []string
	Artifacts   []IRArtifact
	ToolRuns    []IRToolRun
	FilesEdited []string
	RawEvents   []RawEvent
	Notes       []string
}

// IRArtifact captures structured session artifacts in the IR layer.
type IRArtifact struct {
	Name       string
	Type       string
	Content    string
	Attributes map[string]string
}

// IRToolRun captures tool execution telemetry in the IR layer.
type IRToolRun struct {
	ToolName   string
	Command    string
	ExitCode   int
	Output     string
	Attributes map[string]string
}

// RawEvent preserves unknown or extension event types without failing ingestion.
type RawEvent struct {
	Type       string
	Payload    map[string]interface{}
	Attributes map[string]string
	RawLine    string
}

// NewSessionIR returns a baseline session IR for Antigravity ingestion.
func NewSessionIR() *SessionIR {
	return &SessionIR{
		SessionID:   fmt.Sprintf("ag-%d", time.Now().Unix()),
		Agent:       "antigravity",
		Timestamp:   time.Now().UTC(),
		UserPrompts: []string{},
		Artifacts:   []IRArtifact{},
		ToolRuns:    []IRToolRun{},
		FilesEdited: []string{},
		RawEvents:   []RawEvent{},
		Notes:       []string{},
	}
}

// MarkPartial records incompleteness without aborting ingestion.
func (s *SessionIR) MarkPartial(note string) {
	s.Partial = true
	if note != "" {
		s.Notes = append(s.Notes, note)
	}
}

// MergeSessionIR combines two session IR snapshots into one canonical view.
func MergeSessionIR(base, overlay *SessionIR) *SessionIR {
	if base == nil {
		base = NewSessionIR()
	}
	if overlay == nil {
		return base
	}

	if overlay.SessionID != "" {
		base.SessionID = overlay.SessionID
	}
	if overlay.Agent != "" {
		base.Agent = overlay.Agent
	}
	if !overlay.Timestamp.IsZero() {
		base.Timestamp = overlay.Timestamp
	}
	if overlay.Partial {
		base.Partial = true
	}

	base.UserPrompts = appendUniqueStrings(base.UserPrompts, overlay.UserPrompts)
	base.Artifacts = appendUniqueArtifacts(base.Artifacts, overlay.Artifacts)
	base.ToolRuns = appendUniqueToolRuns(base.ToolRuns, overlay.ToolRuns)
	base.FilesEdited = appendUniqueStrings(base.FilesEdited, overlay.FilesEdited)
	base.RawEvents = append(base.RawEvents, overlay.RawEvents...)
	base.Notes = appendUniqueStrings(base.Notes, overlay.Notes)

	return base
}

// ToEntireTranscript projects the canonical IR into the Entire checkpoint schema.
func (s *SessionIR) ToEntireTranscript() *EntireTranscript {
	if s == nil {
		return &EntireTranscript{
			Agent:       "antigravity",
			Timestamp:   time.Now().UTC(),
			UserPrompts: []string{},
			Artifacts:   []CapturedArtifact{},
			ToolRuns:    []CapturedToolRun{},
			FilesEdited: []string{},
		}
	}

	transcript := &EntireTranscript{
		SessionID:   s.SessionID,
		Agent:       s.Agent,
		Timestamp:   s.Timestamp,
		UserPrompts: append([]string{}, s.UserPrompts...),
		FilesEdited: append([]string{}, s.FilesEdited...),
		Artifacts:   make([]CapturedArtifact, 0, len(s.Artifacts)),
		ToolRuns:    make([]CapturedToolRun, 0, len(s.ToolRuns)),
	}

	for _, artifact := range s.Artifacts {
		transcript.Artifacts = append(transcript.Artifacts, CapturedArtifact{
			Name:    artifact.Name,
			Type:    artifact.Type,
			Content: artifact.Content,
		})
	}

	for _, run := range s.ToolRuns {
		transcript.ToolRuns = append(transcript.ToolRuns, CapturedToolRun{
			ToolName: run.ToolName,
			Command:  run.Command,
			ExitCode: run.ExitCode,
			Output:   run.Output,
		})
	}

	return transcript
}

func appendUniqueStrings(dst, src []string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[item] = struct{}{}
	}
	for _, item := range src {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func appendUniqueArtifacts(dst, src []IRArtifact) []IRArtifact {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[artifactKey(item)] = struct{}{}
	}
	for _, item := range src {
		key := artifactKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func appendUniqueToolRuns(dst, src []IRToolRun) []IRToolRun {
	seen := make(map[string]struct{}, len(dst))
	for _, item := range dst {
		seen[toolRunKey(item)] = struct{}{}
	}
	for _, item := range src {
		key := toolRunKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

func artifactKey(artifact IRArtifact) string {
	return artifact.Name + "|" + artifact.Type + "|" + artifact.Content
}

func toolRunKey(run IRToolRun) string {
	return run.ToolName + "|" + run.Command + "|" + run.Output + "|" + fmt.Sprintf("%d", run.ExitCode)
}
