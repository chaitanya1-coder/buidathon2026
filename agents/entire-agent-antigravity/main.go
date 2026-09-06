package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "info":
		if err := handleInfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error handling info: %v\n", err)
			os.Exit(1)
		}

	case "install-hooks":
		installCmd := flag.NewFlagSet("install-hooks", flag.ExitOnError)
		dirFlag := installCmd.String("dir", ".", "Target project directory for configuration")
		_ = installCmd.Parse(os.Args[2:])
		if err := handleInstallHooks(*dirFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing hooks: %v\n", err)
			os.Exit(1)
		}

	case "are-hooks-installed":
		checkCmd := flag.NewFlagSet("are-hooks-installed", flag.ExitOnError)
		dirFlag := checkCmd.String("dir", ".", "Target project directory for configuration")
		_ = checkCmd.Parse(os.Args[2:])
		if err := handleAreHooksInstalled(*dirFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error checking hooks: %v\n", err)
			os.Exit(1)
		}

	case "transcript":
		transcriptCmd := flag.NewFlagSet("transcript", flag.ExitOnError)
		sessionFlag := transcriptCmd.String("session", "", "Session ID or path")
		_ = transcriptCmd.Parse(os.Args[2:])

		// Fallback if --session was not supplied via flag syntax or positional parameter
		sessionID := *sessionFlag
		if sessionID == "" && transcriptCmd.NArg() > 0 {
			sessionID = transcriptCmd.Arg(0)
		}

		if sessionID == "" {
			fmt.Fprintf(os.Stderr, "Error: --session <id> flag is required for transcript subcommand\n")
			os.Exit(1)
		}

		if err := handleTranscript(sessionID); err != nil {
			fmt.Fprintf(os.Stderr, "Error getting transcript: %v\n", err)
			os.Exit(1)
		}

	case "-h", "--help", "help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("entire-agent-antigravity - Entire CLI adapter for Antigravity Agent\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  entire-agent-antigravity <subcommand> [flags]\n\n")
	fmt.Printf("Subcommands:\n")
	fmt.Printf("  info                  Returns agent capabilities JSON\n")
	fmt.Printf("  install-hooks         Injects session hooks into .antigravity/ directory\n")
	fmt.Printf("  are-hooks-installed   Returns {\"installed\": true} if hooks configured\n")
	fmt.Printf("  transcript            Exports conversation transcript for --session <id>\n")
}
