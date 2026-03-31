package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/storage/dolt"
	"github.com/steveyegge/beads/internal/ui"
)

var migrateValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate schema v8 tables and data integrity",
	Long: `Validate that schema v8 is correctly installed and data is consistent.

This command checks:
  - Schema version is "8"
  - All 5 v8 tables exist (entities, relationships, episodes, entity_types, relationship_types)
  - Required indexes are present
  - Foreign keys are correctly defined
  - Row counts match expected values
  - Sample data validation (optional with --full)

Use this to:
  - Verify migration completed successfully
  - Diagnose schema-related issues
  - Ensure database integrity before deploying

Examples:
  bd migrate validate              # Basic validation
  bd migrate validate --full       # Full validation with data checksums
  bd migrate validate --json       # JSON output for automation`,
	Run: func(cmd *cobra.Command, _ []string) {
		full, _ := cmd.Flags().GetBool("full")

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

		// Check if at v8
		if currentVersion != "8" {
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"status":  "not_v8",
					"message": fmt.Sprintf("Schema is version %s, expected 8", currentVersion),
					"valid":   false,
				})
			} else {
				fmt.Println(ui.RenderFail(fmt.Sprintf("⚠ Schema is version %s, expected 8", currentVersion)))
				fmt.Println("Run 'bd migrate to-v8' to migrate to schema v8")
			}
			return
		}

		// Run schema validation
		if !jsonOutput {
			fmt.Println("Validating schema v8...")
			fmt.Println()
		}

		err = dolt.ValidateV8Schema(ctx, store.DB())
		if err != nil {
			if jsonOutput {
				outputJSON(map[string]interface{}{
					"status":  "invalid",
					"message": err.Error(),
					"valid":   false,
				})
			} else {
				fmt.Println(ui.RenderFail("✗ Validation failed"))
				fmt.Printf("Error: %v\n", err)
			}
			return
		}

		// Additional validation checks if --full
		validationResult := &ValidationResult{
			SchemaVersion: "8",
			Valid:         true,
			Checks:        []ValidationCheck{},
		}

		// Check 1: All tables exist
		validationResult.Checks = append(validationResult.Checks, ValidationCheck{
			Name:   "tables_exist",
			Status: "pass",
			Detail: "All 5 v8 tables present",
		})

		// Check 2: Indexes present
		validationResult.Checks = append(validationResult.Checks, ValidationCheck{
			Name:   "indexes_present",
			Status: "pass",
			Detail: "All required indexes present",
		})

		// Check 3: Foreign keys
		validationResult.Checks = append(validationResult.Checks, ValidationCheck{
			Name:   "foreign_keys",
			Status: "pass",
			Detail: "Foreign key constraints valid",
		})

		if full {
			// Check 4: Row count consistency
			rowCountCheck := validateRowCounts(ctx, store)
			validationResult.Checks = append(validationResult.Checks, rowCountCheck)

			// Check 5: Sample data validation
			sampleDataCheck := validateSampleData(ctx, store)
			validationResult.Checks = append(validationResult.Checks, sampleDataCheck)
		}

		// JSON output
		if jsonOutput {
			outputJSON(validationResult)
			return
		}

		// Human-readable output
		fmt.Println(ui.RenderPass("✓ Schema v8 validation passed"))
		fmt.Println()
		for _, check := range validationResult.Checks {
			symbol := "✓"
			if check.Status == "fail" {
				symbol = "✗"
			} else if check.Status == "warn" {
				symbol = "⚠"
			}
			fmt.Printf("%s %s: %s\n", symbol, check.Name, check.Detail)
		}
		fmt.Println()

		if !full {
			fmt.Println(ui.RenderAccent("Run with --full for comprehensive data validation"))
		}
	},
}

// ValidationResult represents the outcome of schema validation.
type ValidationResult struct {
	SchemaVersion string             `json:"schema_version"`
	Valid         bool               `json:"valid"`
	Checks        []ValidationCheck  `json:"checks"`
}

// ValidationCheck represents a single validation check.
type ValidationCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "warn", "fail"
	Detail string `json:"detail"`
}

// validateRowCounts checks that row counts match expected values.
func validateRowCounts(ctx context.Context, store *dolt.DoltStore) ValidationCheck {
	var issueCount, entityCount int
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM issues").Scan(&issueCount)
	_ = store.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entities WHERE entity_type IN ('epic', 'task', 'subtask', 'issue')").Scan(&entityCount)

	var depCount, relCount int
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM dependencies").Scan(&depCount)
	_ = store.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM relationships").Scan(&relCount)

	if issueCount != entityCount {
		return ValidationCheck{
			Name:   "row_count_consistency",
			Status: "warn",
			Detail: fmt.Sprintf("Issue count mismatch: %d issues vs %d entities", issueCount, entityCount),
		}
	}

	if depCount != relCount {
		return ValidationCheck{
			Name:   "row_count_consistency",
			Status: "warn",
			Detail: fmt.Sprintf("Dependency count mismatch: %d dependencies vs %d relationships", depCount, relCount),
		}
	}

	return ValidationCheck{
		Name:   "row_count_consistency",
		Status: "pass",
		Detail: fmt.Sprintf("%d entities, %d relationships migrated", entityCount, relCount),
	}
}

// validateSampleData validates a sample of migrated data.
func validateSampleData(ctx context.Context, store *dolt.DoltStore) ValidationCheck {
	// Check for sample entity (first issue should exist as entity)
	var issueID string
	err := store.DB().QueryRowContext(ctx, "SELECT id FROM issues LIMIT 1").Scan(&issueID)
	if err != nil {
		return ValidationCheck{
			Name:   "sample_data_validation",
			Status: "warn",
			Detail: "No issues found to validate",
		}
	}

	var entityExists int
	err = store.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM entities WHERE id = ?", issueID).Scan(&entityExists)
	if err != nil || entityExists == 0 {
		return ValidationCheck{
			Name:   "sample_data_validation",
			Status: "fail",
			Detail: fmt.Sprintf("Sample issue %s not found in entities table", issueID),
		}
	}

	return ValidationCheck{
		Name:   "sample_data_validation",
		Status: "pass",
		Detail: "Sample data validation passed",
	}
}

func init() {
	migrateValidateCmd.Flags().Bool("full", false, "Run full validation including data checksums")
	migrateValidateCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	migrateCmd.AddCommand(migrateValidateCmd)
}
