package main

import (
	"github.com/spf13/cobra"
)

// ontologyCmd is the parent command for ontology management
var ontologyCmd = &cobra.Command{
	Use:     "ontology",
	GroupID: "knowledge",
	Short:   "Manage custom type schemas (ontology)",
	Long: `Manage custom type schemas for entities and relationships.

The ontology system allows you to register custom types with JSON schema validation,
enabling domain-specific modeling with structured metadata requirements.

Available Commands:
  register-entity-type       Register a new entity type schema
  register-relationship-type Register a new relationship type schema
  list                       List all registered type schemas

Examples:
  # Register a person entity type
  bd ontology register-entity-type --name person --schema person-schema.json

  # List all registered types
  bd ontology list --json`,
}

func init() {
	rootCmd.AddCommand(ontologyCmd)
}
