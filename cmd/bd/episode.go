package main

import (
	"github.com/spf13/cobra"
)

// episodeCmd is the parent command for episode management.
// Episodes are immutable provenance logs that track raw data ingestion.
var episodeCmd = &cobra.Command{
	Use:     "episode",
	GroupID: "core",
	Short:   "Manage episodes (immutable provenance logs)",
	Long: `Manage episodes - immutable provenance logs that track raw data ingestion.

Episodes are the ground truth provenance layer. Each episode represents a snapshot
of ingested data from a source (e.g., "github", "jira", "manual"). Episodes are
append-only and never modified after creation.

Available commands:
  bd episode create --source <source> --file <raw-data-file> --json
  bd episode list --source <source> --since <time> --json
  bd episode show <id> --json

Examples:
  bd episode create --source github --file raw-webhook.json
  bd episode list --source jira --since "2024-01-01"
  bd episode show ep-abc123`,
}

func init() {
	episodeCmd.AddCommand(episodeCreateCmd)
	episodeCmd.AddCommand(episodeListCmd)
	episodeCmd.AddCommand(episodeShowCmd)
	rootCmd.AddCommand(episodeCmd)
}
