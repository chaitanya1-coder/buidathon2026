package main

import (
	"os"
	"path/filepath"
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

// IngestSession parses Antigravity inputs through the unified SessionIR pipeline.
func IngestSession() (*EntireTranscript, error) {
	root, err := FindWorkspaceRoot()
	if err != nil {
		return nil, err
	}

	session := NewSessionIR()

	v1Parser := &V1Parser{}
	v1Session, err := v1Parser.Parse(root)
	if err != nil {
		session.MarkPartial("v1 parser error: " + err.Error())
	} else {
		session = MergeSessionIR(session, v1Session)
	}

	v2Parser := &V2Parser{}
	v2Session, err := v2Parser.Parse(root)
	if err != nil {
		session.MarkPartial("v2 parser error: " + err.Error())
	} else {
		session = MergeSessionIR(session, v2Session)
	}

	if len(session.UserPrompts) == 0 &&
		len(session.Artifacts) == 0 &&
		len(session.ToolRuns) == 0 &&
		len(session.FilesEdited) == 0 &&
		len(session.RawEvents) == 0 {
		session.MarkPartial("no ingestible session data found in .antigravity")
	}

	return session.ToEntireTranscript(), nil
}
