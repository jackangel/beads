package main

import (
	"github.com/spf13/cobra"
)

// entityCmd is the parent command for entity management
var entityCmd = &cobra.Command{
	Use:     "entity",
	GroupID: "knowledge",
	Short:   "Manage knowledge graph entities",
	Long: `Manage knowledge graph entities in the beads system.

Entities are nodes in the knowledge graph representing any trackable object
such as people, components, documents, or domain concepts that can have relationships.

Available Commands:
  create   Create a new entity
  list     List entities matching filters
  show     Show entity details
  update   Update an existing entity
  delete   Delete an entity`,
}

func init() {
	rootCmd.AddCommand(entityCmd)
}
