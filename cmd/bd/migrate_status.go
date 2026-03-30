package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage/dolt"
	"github.com/steveyegge/beads/internal/ui"
)

var migrateStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"version"},
	Short:   "Show current schema version and migration status",
	Long: `Show the current database schema version and table counts.

This command displays:
  - Current schema version (7 or 8)
  - Row counts for v7 tables (issues, dependencies, events)
  - Row counts for v8 tables (entities, relationships, episodes) if present
  - Migration readiness status

Use this to:
  - Check if migration is needed
  - Verify migration completed successfully
  - Diagnose schema-related issues

Examples:
  bd migrate status              # Human-readable output
  bd migrate status --json       # JSON output for automation`,
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := rootCtx
		store := getStore()

		// Get current schema version
		currentVersion, err := dolt.GetSchemaVersion(ctx, store.DB())
		if err != nil {
			FatalErrorRespectJSON("failed to get current schema version: %v", err)
		}

		// Get table counts
		status := getMigrationStatus(ctx, store, currentVersion)

		// JSON output
		if jsonOutput {
			outputJSON(status)
			return
		}

		// Human-readable output
		fmt.Println()
		fmt.Printf("Schema Version: %s\n", ui.RenderAccent(currentVersion))
		fmt.Println()

		// v7 tables (always present)
		fmt.Println("v7 Tables:")
		fmt.Printf("  issues:        %d rows\n", status.V7Tables.Issues)
		fmt.Printf("  dependencies:  %d rows\n", status.V7Tables.Dependencies)
		fmt.Printf("  events:        %d rows\n", status.V7Tables.Events)
		fmt.Println()

		// v8 tables (if present)
		if currentVersion == "8" {
			fmt.Println("v8 Tables:")
			fmt.Printf("  entities:       %d rows\n", status.V8Tables.Entities)
			fmt.Printf("  relationships:  %d rows\n", status.V8Tables.Relationships)
			fmt.Printf("  episodes:       %d rows\n", status.V8Tables.Episodes)
			fmt.Printf("  entity_types:   %d rows\n", status.V8Tables.EntityTypes)
			fmt.Printf("  relationship_types: %d rows\n", status.V8Tables.RelationshipTypes)
			fmt.Println()
			fmt.Println(ui.RenderPass("✓ v8 migration complete"))
		} else if currentVersion == "7" {
			fmt.Println(ui.RenderWarn("⚠ v8 not yet migrated"))
			fmt.Println("Run 'bd migrate to-v8' to migrate to schema v8")
		} else {
			fmt.Println(ui.RenderFail(fmt.Sprintf("⚠ Unknown schema version: %s", currentVersion)))
		}
		fmt.Println()
	},
}

// MigrationStatus represents the current migration state.
type MigrationStatus struct {
	SchemaVersion string    `json:"schema_version"`
	V7Tables      V7Tables  `json:"v7_tables"`
	V8Tables      *V8Tables `json:"v8_tables,omitempty"`
	Status        string    `json:"status"`
}

// V7Tables represents row counts for v7 schema tables.
type V7Tables struct {
	Issues       int `json:"issues"`
	Dependencies int `json:"dependencies"`
	Events       int `json:"events"`
}

// V8Tables represents row counts for v8 schema tables.
type V8Tables struct {
	Entities          int `json:"entities"`
	Relationships     int `json:"relationships"`
	Episodes          int `json:"episodes"`
	EntityTypes       int `json:"entity_types"`
	RelationshipTypes int `json:"relationship_types"`
}

// getMigrationStatus retrieves current migration status with table counts.
func getMigrationStatus(ctx context.Context, store *dolt.DoltStore, currentVersion string) *MigrationStatus {
	status := &MigrationStatus{
		SchemaVersion: currentVersion,
	}

	// Count v7 tables (always present)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&status.V7Tables.Issues)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM dependencies").Scan(&status.V7Tables.Dependencies)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&status.V7Tables.Events)

	// Count v8 tables (if version 8)
	if currentVersion == "8" {
		status.V8Tables = &V8Tables{}
		_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM entities").Scan(&status.V8Tables.Entities)
		_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM relationships").Scan(&status.V8Tables.Relationships)
		_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM episodes").Scan(&status.V8Tables.Episodes)
		_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM entity_types").Scan(&status.V8Tables.EntityTypes)
		_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM relationship_types").Scan(&status.V8Tables.RelationshipTypes)
		status.Status = "v8_migrated"
	} else if currentVersion == "7" {
		status.Status = "v7_ready_to_migrate"
	} else {
		status.Status = "unknown_version"
	}

	return status
}

func init() {
	migrateStatusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	migrateCmd.AddCommand(migrateStatusCmd)
}
