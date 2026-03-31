package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage/dolt"
	"github.com/steveyegge/beads/internal/ui"
)

var migrateRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback from schema v8 to v7",
	Long: `Rollback the database schema from version 8 back to version 7.

This operation:
  - Drops all v8 knowledge graph tables (entities, relationships, episodes, entity_types, relationship_types)
  - Updates schema_version back to "7"
  - Is fully transactional (all-or-nothing)

WARNING: This will DELETE all data in the v8 tables. The v7 tables (issues, dependencies, events)
are preserved and remain unchanged.

This command is intended for:
  - Reverting a failed or problematic migration
  - Testing migration workflows
  - Emergency recovery scenarios

Examples:
  bd migrate rollback              # Rollback with confirmation prompt
  bd migrate rollback --json       # JSON output for automation
  bd migrate rollback --force      # Skip confirmation prompt`,
	Run: func(cmd *cobra.Command, _ []string) {
		force, _ := cmd.Flags().GetBool("force")

		// Block writes in readonly mode
		CheckReadonly("migrate rollback")

		// Initialize store
		if err := ensureStoreActive(); err != nil {
			FatalError("%v", err)
		}

		ctx := rootCtx
		store := getStore()

		// Get current schema version
		currentVersion, err := dolt.GetSchemaVersion(ctx, store.DB())
		if err != nil {
			FatalErrorRespectJSON("failed to get current schema version: %v", err)
		}

		// Check if already at v7
		if currentVersion == "7" {
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"status":  "already_v7",
					"message": "Schema is already at version 7",
					"version": "7",
				})
			} else {
				fmt.Println(ui.RenderWarn("⚠ Schema is already at version 7"))
				fmt.Println("Nothing to do.")
			}
			return
		}

		// Check if coming from v8
		if currentVersion != "8" {
			FatalErrorRespectJSON("rollback from v8 requires schema v8, but found v%s", currentVersion)
		}

		// Confirmation prompt (unless --force or --json)
		if !force && !jsonOutput {
			fmt.Println(ui.RenderFail("⚠ WARNING: This will DELETE all data in v8 tables"))
			fmt.Println()
			fmt.Println("The following tables will be dropped:")
			fmt.Println("  - entities")
			fmt.Println("  - relationships")
			fmt.Println("  - episodes")
			fmt.Println("  - entity_types")
			fmt.Println("  - relationship_types")
			fmt.Println()
			fmt.Println("v7 tables (issues, dependencies, events) will remain unchanged.")
			fmt.Println()

			if !confirmPrompt("This will revert to v7. Continue? (y/N)") {
				if !jsonOutput {
					fmt.Println("Rollback cancelled.")
				}
				return
			}
		}

		// Run the rollback
		if !jsonOutput {
			fmt.Println("Starting rollback to schema v7...")
		}

		err = dolt.RollbackFromV8(ctx, store.DB())
		if err != nil {
			FatalErrorRespectJSON("rollback failed: %v", err)
		}

		// Success output
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"message": "Rollback to schema v7 completed successfully",
				"version": "7",
			})
		} else {
			fmt.Println()
			fmt.Println(ui.RenderPass("✓ Rollback to schema v7 completed successfully"))
			fmt.Println()
			fmt.Println("Dropped tables:")
			fmt.Println("  - entities")
			fmt.Println("  - relationships")
			fmt.Println("  - episodes")
			fmt.Println("  - entity_types")
			fmt.Println("  - relationship_types")
		}
	},
}

func init() {
	migrateRollbackCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	migrateRollbackCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	migrateCmd.AddCommand(migrateRollbackCmd)
}
