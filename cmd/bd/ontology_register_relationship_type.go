package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/types"
)

var ontologyRegisterRelationshipTypeCmd = &cobra.Command{
	Use:   "register-relationship-type",
	Short: "Register a new relationship type schema",
	Long: `Register a new relationship type with JSON schema validation.

Relationship type schemas define the structure, required fields, and validation rules
for relationships of a specific type (e.g., "uses", "implements", "replaces").

The schema file must contain valid JSON Schema (draft-07 or later) that will be
used to validate relationship metadata when creating or updating relationships of this type.

Examples:
  # Register a "uses" relationship type
  bd ontology register-relationship-type --name uses --schema uses-schema.json

  # Register with description
  bd ontology register-relationship-type --name implements --schema impl.json --description "Implementation relationship"

  # Register with JSON output
  bd ontology register-relationship-type --name replaces --schema replace.json --json`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("ontology register-relationship-type")
		ctx := rootCtx

		// Get required flags
		typeName, _ := cmd.Flags().GetString("name")
		if typeName == "" {
			FatalErrorRespectJSON("--name is required")
		}

		schemaFile, _ := cmd.Flags().GetString("schema")
		if schemaFile == "" {
			FatalErrorRespectJSON("--schema is required")
		}

		// Get optional flags
		description, _ := cmd.Flags().GetString("description")
		createdBy, _ := cmd.Flags().GetString("created-by")
		if createdBy == "" {
			createdBy = getActorWithGit()
		}

		// Read schema file
		schemaBytes, err := os.ReadFile(schemaFile)
		if err != nil {
			FatalErrorRespectJSON("failed to read schema file %s: %v", schemaFile, err)
		}

		// Validate that the schema is valid JSON
		var schemaValidation interface{}
		if err := json.Unmarshal(schemaBytes, &schemaValidation); err != nil {
			FatalErrorRespectJSON("invalid JSON in schema file %s: %v", schemaFile, err)
		}

		// Create relationship type schema
		schema := &types.RelationshipTypeSchema{
			TypeName:    typeName,
			SchemaJSON:  string(schemaBytes),
			Description: description,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			CreatedBy:   createdBy,
		}

		// Register the schema
		err = store.RegisterRelationshipType(ctx, schema)
		if err != nil {
			FatalErrorRespectJSON("failed to register relationship type: %v", err)
		}

		// Output result
		if jsonOutput {
			output := map[string]interface{}{
				"type_name":   schema.TypeName,
				"description": schema.Description,
				"schema_json": schema.SchemaJSON,
				"created_at":  schema.CreatedAt,
				"created_by":  schema.CreatedBy,
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Fprintf(os.Stderr, "Registered relationship type: %s\n", schema.TypeName)
			if schema.Description != "" {
				fmt.Printf("Description: %s\n", schema.Description)
			}
			fmt.Printf("Schema: %s\n", schemaFile)
		}
	},
}

func init() {
	ontologyCmd.AddCommand(ontologyRegisterRelationshipTypeCmd)

	ontologyRegisterRelationshipTypeCmd.Flags().String("name", "", "Relationship type name (e.g., uses, implements, replaces)")
	ontologyRegisterRelationshipTypeCmd.Flags().String("schema", "", "Path to JSON schema file")
	ontologyRegisterRelationshipTypeCmd.Flags().String("description", "", "Human-readable description of this relationship type")
	ontologyRegisterRelationshipTypeCmd.Flags().String("created-by", "", "Creator name (defaults to current actor)")
}
