package main

import (
	"github.com/spf13/cobra"
)

// relationshipCmd is the parent command for relationship management.
// Relationships represent typed, directional edges in the knowledge graph with temporal validity.
var relationshipCmd = &cobra.Command{
	Use:     "relationship",
	Aliases: []string{"rel", "relationships"},
	GroupID: "deps",
	Short:   "Manage relationships between entities",
	Long: `Manage relationships in the knowledge graph.

Relationships are typed, directional edges between entities with temporal validity.
They track how connections evolve over time with ValidFrom and ValidUntil timestamps.

Examples:
  # Create a relationship
  bd relationship create --from entity-1 --type uses --to entity-2

  # List outgoing relationships
  bd relationship list --from entity-1

  # List incoming relationships
  bd relationship list --to entity-1

  # Show relationship details
  bd relationship show rel-abc123

  # Update relationship (close temporal window)
  bd relationship update rel-abc123 --valid-until "2024-12-31 23:59"

  # Delete relationship
  bd relationship delete rel-abc123`,
}

func init() {
	rootCmd.AddCommand(relationshipCmd)
}
