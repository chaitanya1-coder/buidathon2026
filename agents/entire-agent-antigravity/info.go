package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Features struct {
	Hooks             bool `json:"hooks"`
	Transcripts       bool `json:"transcripts"`
	CompactTranscripts bool `json:"compact_transcripts"`
}

type AgentInfo struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Features Features `json:"features"`
}

func handleInfo() error {
	agentName := "antigravity"
	if strings.Contains(filepath.Base(os.Args[0]), "opencode") {
		agentName = "opencode"
	}

	info := AgentInfo{
		Name:    agentName,
		Version: "0.1.0",
		Features: Features{
			Hooks:             true,
			Transcripts:       true,
			CompactTranscripts: true,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}
