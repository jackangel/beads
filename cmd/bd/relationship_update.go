package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/timeparsing"
	"github.com/steveyegge/beads/internal/types"
)

var relationshipUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a relationship's temporal validity",
	Long: `Update a relationship's temporal validity window.

This command is primarily used to close the temporal window by setting --valid-until,
or to extend a relationship by modifying --valid-from or --valid-until.

For most use cases, you'll want to set --valid-until to mark when a relationship ended.
This preserves the historical record while marking the relationship as no longer active.

Examples:
  # Close temporal window (mark relationship as ended)
  bd relationship update rel-abc123 --valid-until "2024-12-31 23:59"

  # Extend relationship validity
  bd relationship update rel-abc123 --valid-until "2025-12-31"

  # Change validity start (use with caution - affects historical queries)
  bd relationship update rel-abc123 --valid-from "2024-01-01"

  # Update metadata
  bd relationship update rel-abc123 --metadata '{"status":"archived"}'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("relationship update")
		ctx := rootCtx
		relID := args[0]

		// Get flag values
		validFromStr, _ := cmd.Flags().GetString("valid-from")
		validUntilStr, _ := cmd.Flags().GetString("valid-until")
		metadataStr, _ := cmd.Flags().GetString("metadata")

		// Validate that at least one field is being updated
		if validFromStr == "" && validUntilStr == "" && metadataStr == "" {
			FatalErrorRespectJSON("at least one of --valid-from, --valid-until, or --metadata must be specified")
		}

		// Retrieve existing relationship to validate
		store := getStore()
		existing, err := store.GetRelationship(ctx, relID)
		if err != nil {
			FatalErrorRespectJSON("failed to get relationship: %v", err)
		}

		// Build update object with only specified fields
		update := &types.Relationship{
			ID: relID,
		}

		// Parse valid-from
		if validFromStr != "" {
			parsed, err := timeparsing.ParseRelativeTime(validFromStr, time.Now())
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-from: %v", err)
			}
			update.ValidFrom = parsed
		}

		// Parse valid-until
		if validUntilStr != "" {
			parsed, err := timeparsing.ParseRelativeTime(validUntilStr, time.Now())
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-until: %v", err)
			}
			update.ValidUntil = &parsed

			// Validate temporal window
			validFrom := existing.ValidFrom
			if !update.ValidFrom.IsZero() {
				validFrom = update.ValidFrom
			}
			if update.ValidUntil.Before(validFrom) {
				FatalErrorRespectJSON("--valid-until must be after --valid-from")
			}
		}

		// Parse metadata
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &update.Metadata); err != nil {
				FatalErrorRespectJSON("invalid --metadata JSON: %v", err)
			}
		}

		// Update relationship
		if err := store.UpdateRelationship(ctx, update); err != nil {
			FatalErrorRespectJSON("failed to update relationship: %v", err)
		}

		// Mark that we wrote data
		commandDidWrite.Store(true)

		// Retrieve updated relationship for display
		updated, err := store.GetRelationship(ctx, relID)
		if err != nil {
			FatalErrorRespectJSON("failed to retrieve updated relationship: %v", err)
		}

		// Output result
		if jsonOutput {
			data, err := json.MarshalIndent(updated, "", "  ")
			if err != nil {
				FatalErrorRespectJSON("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Printf("Updated relationship %s\n", relID)
			if validFromStr != "" {
				fmt.Printf("  Valid from: %s\n", updated.ValidFrom.Format("2006-01-02 15:04"))
			}
			if validUntilStr != "" {
				fmt.Printf("  Valid until: %s\n", updated.ValidUntil.Format("2006-01-02 15:04"))
			}
			if metadataStr != "" {
				fmt.Printf("  Metadata updated\n")
			}
		}
	},
}

func init() {
	relationshipUpdateCmd.Flags().String("valid-from", "", "Update start of validity window")
	relationshipUpdateCmd.Flags().String("valid-until", "", "Update end of validity window")
	relationshipUpdateCmd.Flags().String("metadata", "", "Update metadata as JSON object")

	relationshipCmd.AddCommand(relationshipUpdateCmd)
}
