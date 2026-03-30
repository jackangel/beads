package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var ontologyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered type schemas",
	Long: `List all registered entity and relationship type schemas.

Shows all custom types that have been registered in the ontology, including
entity types (e.g., "person", "component") and relationship types (e.g., "uses", "implements").

Examples:
  # List all registered types
  bd ontology list

  # List with JSON output
  bd ontology list --json`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := rootCtx

		// Get all entity types
		entityTypes, err := store.GetEntityTypes(ctx)
		if err != nil {
			FatalErrorRespectJSON("failed to get entity types: %v", err)
		}

		// Get all relationship types
		relationshipTypes, err := store.GetRelationshipTypes(ctx)
		if err != nil {
			FatalErrorRespectJSON("failed to get relationship types: %v", err)
		}

		// Output result
		if jsonOutput {
			output := map[string]interface{}{
				"entity_types":       entityTypes,
				"relationship_types": relationshipTypes,
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			// Human-readable output
			if len(entityTypes) > 0 {
				fmt.Fprintln(os.Stderr, "Entity Types:")
				for _, et := range entityTypes {
					fmt.Printf("  %s", et.TypeName)
					if et.Description != "" {
						fmt.Printf(" - %s", et.Description)
					}
					fmt.Println()
				}
			} else {
				fmt.Fprintln(os.Stderr, "No entity types registered.")
			}

			fmt.Println()

			if len(relationshipTypes) > 0 {
				fmt.Fprintln(os.Stderr, "Relationship Types:")
				for _, rt := range relationshipTypes {
					fmt.Printf("  %s", rt.TypeName)
					if rt.Description != "" {
						fmt.Printf(" - %s", rt.Description)
					}
					fmt.Println()
				}
			} else {
				fmt.Fprintln(os.Stderr, "No relationship types registered.")
			}

			// Summary

			fmt.Fprintf(os.Stderr, "\nTotal: %d entity types, %d relationship types\n",
				len(entityTypes), len(relationshipTypes))
		}
	},
}

func init() {
	ontologyCmd.AddCommand(ontologyListCmd)
}
