package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestV1ParserIngestsLegacyFixture(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, ".antigravity")); err != nil {
		t.Skip("demo fixture not present")
	}

	session, err := (&V1Parser{}).Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(session.UserPrompts) == 0 {
		t.Fatalf("expected user prompt from legacy fixture")
	}
	if len(session.Artifacts) < 2 {
		t.Fatalf("expected artifacts from legacy fixture, got %d", len(session.Artifacts))
	}
	if len(session.ToolRuns) < 2 {
		t.Fatalf("expected tool runs from legacy fixture, got %d", len(session.ToolRuns))
	}
}

func TestV2ParserStoresUnknownEvents(t *testing.T) {
	dir := t.TempDir()
	agDir := filepath.Join(dir, ".antigravity")
	if err := os.MkdirAll(agDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	events := `{"type":"user_prompt","text":"ship it"}
{"type":"custom_metric","value":42}
`
	if err := os.WriteFile(filepath.Join(agDir, "session.jsonl"), []byte(events), 0644); err != nil {
		t.Fatalf("write session.jsonl: %v", err)
	}

	session, err := (&V2Parser{}).Parse(dir)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(session.UserPrompts) != 1 || session.UserPrompts[0] != "ship it" {
		t.Fatalf("unexpected prompts: %#v", session.UserPrompts)
	}
	if len(session.RawEvents) != 1 {
		t.Fatalf("expected unknown event preserved, got %#v", session.RawEvents)
	}
	if session.RawEvents[0].Type != "custom_metric" {
		t.Fatalf("unexpected raw event type: %s", session.RawEvents[0].Type)
	}
}

func TestMergeSessionIRMarksPartial(t *testing.T) {
	base := NewSessionIR()
	overlay := NewSessionIR()
	overlay.MarkPartial("missing execution log")

	merged := MergeSessionIR(base, overlay)
	if !merged.Partial {
		t.Fatalf("expected merged session to remain partial")
	}
	if len(merged.Notes) == 0 {
		t.Fatalf("expected partial notes to be preserved")
	}
}

func TestSessionIRToEntireTranscript(t *testing.T) {
	session := NewSessionIR()
	session.UserPrompts = []string{"prompt"}
	session.Artifacts = []IRArtifact{{Name: "plan.md", Type: "plan", Content: "body"}}
	session.ToolRuns = []IRToolRun{{ToolName: "bash", Command: "npm test", ExitCode: 0, Output: "ok"}}

	transcript := session.ToEntireTranscript()
	if len(transcript.UserPrompts) != 1 || len(transcript.Artifacts) != 1 || len(transcript.ToolRuns) != 1 {
		t.Fatalf("unexpected transcript projection: %#v", transcript)
	}
}
