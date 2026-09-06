package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// V2Parser ingests unified Antigravity event-stream JSONL.
type V2Parser struct{}

func (p *V2Parser) Parse(root string) (*SessionIR, error) {
	session := NewSessionIR()
	eventFile := filepath.Join(root, ".antigravity", "session.jsonl")

	file, err := os.Open(eventFile)
	if err != nil {
		if os.IsNotExist(err) {
			return session, nil
		}
		session.MarkPartial("failed to open session.jsonl: " + err.Error())
		return session, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	parsedEvents := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			session.MarkPartial("skipped malformed session.jsonl line")
			session.RawEvents = append(session.RawEvents, RawEvent{
				Type:    "malformed_jsonl",
				RawLine: line,
				Payload: map[string]interface{}{"parse_error": err.Error()},
			})
			continue
		}

		parsedEvents++
		p.applyEvent(session, event, line)
	}
	if err := scanner.Err(); err != nil {
		session.MarkPartial("session.jsonl scan error: " + err.Error())
	}
	if parsedEvents == 0 {
		session.MarkPartial("session.jsonl contained no parseable events")
	}

	return session, nil
}

func (p *V2Parser) applyEvent(session *SessionIR, event map[string]interface{}, rawLine string) {
	eventType := strings.TrimSpace(asString(event["type"]))
	if eventType == "" {
		session.MarkPartial("event missing type field")
		session.RawEvents = append(session.RawEvents, RawEvent{
			Type:    "missing_type",
			RawLine: rawLine,
			Payload: event,
		})
		return
	}

	switch eventType {
	case "user_prompt", "prompt":
		if prompt := firstNonEmpty(asString(event["text"]), asString(event["prompt"]), asString(event["content"])); prompt != "" {
			session.UserPrompts = append(session.UserPrompts, prompt)
		}
	case "artifact_update", "artifact":
		artifact := IRArtifact{
			Name:    firstNonEmpty(asString(event["name"]), asString(event["artifact_name"])),
			Type:    firstNonEmpty(asString(event["artifact_type"]), asString(event["type_hint"])),
			Content: firstNonEmpty(asString(event["content"]), asString(event["body"])),
		}
		if artifact.Name == "" && artifact.Content == "" {
			session.MarkPartial("artifact_update event missing name and content")
			break
		}
		if artifact.Type == "" {
			artifact.Type = artifactTypeFromName(artifact.Name)
		}
		session.Artifacts = append(session.Artifacts, artifact)
	case "tool_call", "tool_result", "tool_run":
		run := IRToolRun{
			ToolName: firstNonEmpty(asString(event["tool"]), asString(event["tool_name"])),
			Command:  firstNonEmpty(asString(event["command"]), asString(event["input"])),
			Output:   firstNonEmpty(asString(event["output"]), asString(event["result"])),
			ExitCode: asInt(event["exit_code"]),
		}
		if run.ToolName == "" && run.Command == "" && run.Output == "" {
			session.MarkPartial("tool event missing actionable fields")
			break
		}
		session.ToolRuns = append(session.ToolRuns, run)
	case "file_edit", "file_changed":
		if path := firstNonEmpty(asString(event["path"]), asString(event["file"]), asString(event["target_file"])); path != "" {
			session.FilesEdited = append(session.FilesEdited, path)
		}
	case "session_metadata":
		if sessionID := asString(event["session_id"]); sessionID != "" {
			session.SessionID = sessionID
		}
		if agent := asString(event["agent"]); agent != "" {
			session.Agent = agent
		}
	default:
		session.RawEvents = append(session.RawEvents, RawEvent{
			Type:       eventType,
			Payload:    event,
			Attributes: stringMapFromEvent(event),
			RawLine:    rawLine,
		})
	}
}

func asString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func asInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func stringMapFromEvent(event map[string]interface{}) map[string]string {
	attrs := make(map[string]string)
	for key, value := range event {
		if str := asString(value); str != "" {
			attrs[key] = str
		}
	}
	return attrs
}
