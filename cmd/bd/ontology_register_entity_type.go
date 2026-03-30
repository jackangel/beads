package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/types"
)

var ontologyRegisterEntityTypeCmd = &cobra.Command{
	Use:   "register-entity-type",
	Short: "Register a new entity type schema",
	Long: `Register a new entity type with JSON schema validation.

Entity type schemas define the structure, required fields, and validation rules
for entities of a specific type (e.g., "person", "component", "document").

The schema file must contain valid JSON Schema (draft-07 or later) that will be
used to validate entity metadata when creating or updating entities of this type.

Examples:
  # Register a person entity type
  bd ontology register-entity-type --name person --schema person-schema.json

  # Register with description
  bd ontology register-entity-type --name component --schema component.json --description "Software component type"

  # Register with JSON output
  bd ontology register-entity-type --name product --schema product.json --json`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("ontology register-entity-type")
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

		// Create entity type schema
		schema := &types.EntityTypeSchema{
			TypeName:    typeName,
			SchemaJSON:  string(schemaBytes),
			Description: description,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			CreatedBy:   createdBy,
		}

		// Register the schema
		err = store.RegisterEntityType(ctx, schema)
		if err != nil {
			FatalErrorRespectJSON("failed to register entity type: %v", err)
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
			fmt.Fprintf(os.Stderr, "Registered entity type: %s\n", schema.TypeName)
			if schema.Description != "" {
				fmt.Printf("Description: %s\n", schema.Description)
			}
			fmt.Printf("Schema: %s\n", schemaFile)
		}
	},
}

func init() {
	ontologyCmd.AddCommand(ontologyRegisterEntityTypeCmd)

	ontologyRegisterEntityTypeCmd.Flags().String("name", "", "Entity type name (e.g., person, component, document)")
	ontologyRegisterEntityTypeCmd.Flags().String("schema", "", "Path to JSON schema file")
	ontologyRegisterEntityTypeCmd.Flags().String("description", "", "Human-readable description of this entity type")
	ontologyRegisterEntityTypeCmd.Flags().String("created-by", "", "Creator name (defaults to current actor)")
}
