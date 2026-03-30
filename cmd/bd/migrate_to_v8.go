package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage/dolt"
	"github.com/steveyegge/beads/internal/ui"
)

var migrateToV8Cmd = &cobra.Command{
	Use:   "to-v8",
	Short: "Migrate database from schema v7 to v8 (knowledge graph)",
	Long: `Migrate the database from schema version 7 to version 8.

This migration creates 5 new knowledge graph tables:
  - entities: Generic entity storage (migrates from issues)
  - relationships: Generic relationship storage (migrates from dependencies)
  - episodes: Event/episode storage (migrates from events)
  - entity_types: Entity type definitions with JSON schemas
  - relationship_types: Relationship type definitions with JSON schemas

The migration:
  - Creates all new tables with indexes and foreign keys
  - Migrates data from v7 tables (issues → entities, dependencies → relationships, events → episodes)
  - Validates row counts match expected values
  - Updates schema_version to "8"
  - Is fully transactional (all-or-nothing)

WARNING: This migration may take several minutes for large databases.

Examples:
  bd migrate to-v8                    # Run migration
  bd migrate to-v8 --dry-run          # Preview what will happen
  bd migrate to-v8 --dry-run --json   # JSON preview for automation`,
	Run: func(cmd *cobra.Command, _ []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Block writes in readonly mode (unless dry-run)
		if !dryRun {
			CheckReadonly("migrate to-v8")
		}

		ctx := rootCtx
		store := getStore()

		// Dry-run mode: Check current version and show what would happen
		if dryRun {
			handleToV8DryRun(ctx, store)
			return
		}

		// Get current schema version
		currentVersion, err := dolt.GetSchemaVersion(ctx, store.DB())
		if err != nil {
			FatalErrorRespectJSON("failed to get current schema version: %v", err)
		}

		// Check if already at v8
		if currentVersion == "8" {
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"status":  "already_migrated",
					"message": "Schema is already at version 8",
					"version": "8",
				})
			} else {
				fmt.Fprintln(os.Stderr, ui.RenderWarn("⚠ Schema is already at version 8"))
				fmt.Fprintln(os.Stderr, "Nothing to do.")
			}
			return
		}

		// Check if coming from v7
		if currentVersion != "7" {
			FatalErrorRespectJSON("migration to v8 requires schema v7, but found v%s", currentVersion)
		}

		// Show progress message
		if !jsonOutput {
			fmt.Println("Starting migration to schema v8...")
			fmt.Println("This may take several minutes for large databases.")
			fmt.Println()
		}

		// Run the migration
		err = dolt.MigrateToV8(ctx, store.DB())
		if err != nil {
			FatalErrorRespectJSON("migration failed: %v", err)
		}

		// Success output
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"status":  "success",
				"message": "Migration to schema v8 completed successfully",
				"version": "8",
			})
		} else {
			fmt.Println()
			fmt.Println(ui.RenderPass("✓ Migration to schema v8 completed successfully"))
			fmt.Println()
			fmt.Println("New tables created:")
			fmt.Println("  - entities (migrated from issues)")
			fmt.Println("  - relationships (migrated from dependencies)")
			fmt.Println("  - episodes (migrated from events)")
			fmt.Println("  - entity_types")
			fmt.Println("  - relationship_types")
			fmt.Println()
			fmt.Println("Run 'bd migrate validate' to verify the migration.")
		}
	},
}

// handleToV8DryRun shows what the migration would do without applying changes.
func handleToV8DryRun(ctx context.Context, store *dolt.DoltStore) {
	currentVersion, err := dolt.GetSchemaVersion(ctx, store.DB())
	if err != nil {
		FatalErrorRespectJSON("failed to get current schema version: %v", err)
	}

	// Count rows in v7 tables
	var issueCount, depCount, eventCount int
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&issueCount)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM dependencies").Scan(&depCount)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&eventCount)

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"dry_run":         true,
			"current_version": currentVersion,
			"can_migrate":     currentVersion == "7",
			"operations": map[string]interface{}{
				"create_tables": []string{
					"entities",
					"relationships",
					"episodes",
					"entity_types",
					"relationship_types",
				},
				"migrate_data": map[string]interface{}{
					"issues_to_entities":            issueCount,
					"dependencies_to_relationships": depCount,
					"events_to_episodes":            eventCount,
				},
				"update_schema_version": "8",
			},
		})
	} else {
		fmt.Println("DRY RUN - No changes will be made")
		fmt.Println()
		fmt.Printf("Current schema version: %s\n", currentVersion)
		fmt.Println()

		if currentVersion == "8" {
			fmt.Println(ui.RenderWarn("⚠ Schema is already at version 8"))
			fmt.Println("Nothing to do.")
			return
		}

		if currentVersion != "7" {
			fmt.Println(ui.RenderFail(fmt.Sprintf("⚠ Migration requires schema v7, but found v%s", currentVersion)))
			fmt.Println("Cannot migrate.")
			return
		}

		fmt.Println("Migration plan:")
		fmt.Println()
		fmt.Println("1. Create new tables:")
		fmt.Println("   - entities")
		fmt.Println("   - relationships")
		fmt.Println("   - episodes")
		fmt.Println("   - entity_types")
		fmt.Println("   - relationship_types")
		fmt.Println()
		fmt.Println("2. Migrate data:")
		fmt.Printf("   - %d issues → entities\n", issueCount)
		fmt.Printf("   - %d dependencies → relationships\n", depCount)
		fmt.Printf("   - %d events → episodes\n", eventCount)
		fmt.Println()
		fmt.Println("3. Update schema_version to 8")
		fmt.Println()
		fmt.Println(ui.RenderAccent("Run without --dry-run to apply migration"))
	}
}

func init() {
	migrateToV8Cmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	migrateToV8Cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	migrateCmd.AddCommand(migrateToV8Cmd)
}
