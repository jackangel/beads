package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var compatCmd = &cobra.Command{
	Use:     "compat",
	GroupID: "setup",
	Short:   "Manage compatibility mode (v7/v8 schema)",
	Long: `Manage compatibility mode for schema version selection.

beads supports dual-mode operation during the knowledge graph migration:
  - v7: Legacy issue-based schema (issues, dependencies, child_counters)
  - v8: Knowledge graph schema (entities, relationships, episodes)

The compatibility mode determines which schema is active for all operations.
This allows gradual migration from v7 to v8 with rollback capability.

Commands:
  bd compat set v7       - Switch to v7 schema mode (legacy issues)
  bd compat set v8       - Switch to v8 schema mode (knowledge graph)
  bd compat status       - Show current compatibility mode

The mode is persisted in the database configuration and affects all bd commands.

Examples:
  bd compat status
  bd compat set v7
  bd compat set v8
  bd compat status --json`,
}

var compatSetCmd = &cobra.Command{
	Use:   "set <mode>",
	Short: "Set compatibility mode (v7 or v8)",
	Long: `Set the compatibility mode for schema version selection.

Valid modes:
  v7 - Use legacy issue-based schema
  v8 - Use knowledge graph schema (entities, relationships, episodes)

The mode persists across all bd commands until changed again.

Before switching modes, ensure:
  - Target schema has been initialized
  - Data migration is complete (if switching to v8)
  - All team members are ready to use the new mode

Examples:
  bd compat set v7
  bd compat set v8
  bd compat set v8 --json`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		mode := args[0]

		// Validate mode
		if mode != "v7" && mode != "v8" {
			fmt.Fprintf(os.Stderr, "Error: invalid mode %q (valid values: v7, v8)\n", mode)
			os.Exit(1)
		}

		// Database operations require direct mode
		if err := ensureDirectMode("compat set requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx

		// Set compatibility mode in config
		if err := store.SetConfig(ctx, "compat_mode", mode); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting compatibility mode: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			outputJSON(map[string]interface{}{
				"compat_mode": mode,
				"message":     fmt.Sprintf("Switched to %s mode", mode),
			})
		} else {
			fmt.Printf("✓ Switched to %s mode\n", mode)
			if mode == "v7" {
				fmt.Println("\nUsing legacy issue-based schema (issues, dependencies, child_counters)")
			} else {
				fmt.Println("\nUsing knowledge graph schema (entities, relationships, episodes)")
			}
		}
	},
}

var compatStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current compatibility mode",
	Long: `Show the current compatibility mode setting.

Displays which schema version is currently active:
  - v7: Legacy issue-based schema
  - v8: Knowledge graph schema

Examples:
  bd compat status
  bd compat status --json`,
	Run: func(_ *cobra.Command, args []string) {
		// Database operations require direct mode
		if err := ensureDirectMode("compat status requires direct database access"); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := rootCtx

		// Get compatibility mode from config
		mode, err := store.GetConfig(ctx, "compat_mode")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting compatibility mode: %v\n", err)
			os.Exit(1)
		}

		// Default to v7 if not set
		if mode == "" {
			mode = "v7"
		}

		// Validate mode (defensive check)
		if mode != "v7" && mode != "v8" {
			fmt.Fprintf(os.Stderr, "Warning: invalid mode %q in config, defaulting to v7\n", mode)
			mode = "v7"
		}

		if jsonOutput {
			schemaDesc := "legacy issue-based schema"
			if mode == "v8" {
				schemaDesc = "knowledge graph schema"
			}
			outputJSON(map[string]interface{}{
				"compat_mode": mode,
				"schema":      schemaDesc,
			})
		} else {
			fmt.Printf("Current compatibility mode: %s\n", mode)
			if mode == "v7" {
				fmt.Println("\nUsing legacy issue-based schema:")
				fmt.Println("  • issues table")
				fmt.Println("  • dependencies table")
				fmt.Println("  • child_counters table")
			} else {
				fmt.Println("\nUsing knowledge graph schema:")
				fmt.Println("  • entities table")
				fmt.Println("  • relationships table")
				fmt.Println("  • episodes table")
				fmt.Println("  • entity_types table")
				fmt.Println("  • relationship_types table")
			}
		}
	},
}

func init() {
	compatCmd.AddCommand(compatSetCmd)
	compatCmd.AddCommand(compatStatusCmd)
	rootCmd.AddCommand(compatCmd)
}
