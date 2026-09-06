package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStreamV1ToolEvents(t *testing.T) {
	input := strings.NewReader(`{"tool":"bash","command":"npm test","exit_code":1,"output":"fail"}
{"tool":"bash","command":"npm test","exit_code":0,"output":"ok"}
`)
	ir, err := ParseStream(input)
	if err != nil {
		t.Fatalf("ParseStream() error: %v", err)
	}
	if len(ir.ToolRuns) != 2 {
		t.Fatalf("expected 2 tool runs, got %d", len(ir.ToolRuns))
	}
}

func TestParseStreamV2EventsAndUnknownTypes(t *testing.T) {
	input := strings.NewReader(`{"type":"user_prompt","payload":{"prompt":"ship it"}}
{"type":"custom_metric","payload":{"value":42}}
`)
	ir, err := ParseStream(input)
	if err != nil {
		t.Fatalf("ParseStream() error: %v", err)
	}
	if len(ir.UserPrompts) != 1 || ir.UserPrompts[0] != "ship it" {
		t.Fatalf("unexpected prompts: %#v", ir.UserPrompts)
	}
	if len(ir.Warnings) != 1 || !strings.Contains(ir.Warnings[0], "custom_metric") {
		t.Fatalf("expected warning for unknown event, got %#v", ir.Warnings)
	}
}

func TestParseStreamPartialOnMalformedLine(t *testing.T) {
	input := strings.NewReader(`{"tool":"bash","command":"npm test","exit_code":0,"output":"ok"}
{broken
`)
	ir, err := ParseStream(input)
	if err != nil {
		t.Fatalf("ParseStream() error: %v", err)
	}
	if !ir.IsPartial {
		t.Fatalf("expected partial result on malformed stream")
	}
	if len(ir.ToolRuns) != 1 {
		t.Fatalf("expected preserved tool run before malformed line")
	}
}

func TestIngestSessionLegacyFixture(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, ".antigravity")); err != nil {
		t.Skip("demo fixture not present")
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)

	session, err := IngestSession()
	if err != nil {
		t.Fatalf("IngestSession() error: %v", err)
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

func TestSessionIRToEntireTranscript(t *testing.T) {
	ir := &SessionIR{
		SessionID:   "ag-test",
		Agent:       "antigravity",
		UserPrompts: []string{"prompt"},
		Artifacts:   []CapturedArtifact{{Name: "plan.md", Type: "plan", Content: "body"}},
		ToolRuns:    []CapturedToolRun{{ToolName: "bash", Command: "npm test", ExitCode: 0, Output: "ok"}},
	}

	transcript := ir.ToEntireTranscript()
	if len(transcript.UserPrompts) != 1 || len(transcript.Artifacts) != 1 || len(transcript.ToolRuns) != 1 {
		t.Fatalf("unexpected transcript projection: %#v", transcript)
	}
}
