package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/idgen"
	"github.com/steveyegge/beads/internal/timeparsing"
	"github.com/steveyegge/beads/internal/types"
)

var relationshipCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new relationship between entities",
	Long: `Create a new relationship between two entities in the knowledge graph.

Relationships are typed, directional edges with temporal validity. Each relationship
has a source entity (--from), a relationship type (--type), and a target entity (--to).

Temporal validity is specified with --valid-from and --valid-until flags. If --valid-from
is not specified, it defaults to the current time. If --valid-until is not specified,
the relationship is considered valid indefinitely.

Examples:
  # Create a "uses" relationship
  bd relationship create --from component-1 --type uses --to library-1

  # Create with specific validity window
  bd relationship create --from entity-1 --type implements --to entity-2 \
    --valid-from "2024-01-01" --valid-until "2024-12-31"

  # Create with metadata
  bd relationship create --from person-1 --type works-on --to project-1 \
    --metadata '{"role":"lead", "allocation":0.5}'`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckReadonly("relationship create")
		ctx := rootCtx

		// Get flag values
		fromEntity, _ := cmd.Flags().GetString("from")
		toEntity, _ := cmd.Flags().GetString("to")
		relType, _ := cmd.Flags().GetString("type")
		validFromStr, _ := cmd.Flags().GetString("valid-from")
		validUntilStr, _ := cmd.Flags().GetString("valid-until")
		metadataStr, _ := cmd.Flags().GetString("metadata")
		confidence, _ := cmd.Flags().GetFloat64("confidence")

		// Validate required fields
		if fromEntity == "" {
			FatalErrorRespectJSON("--from is required (source entity ID)")
		}
		if toEntity == "" {
			FatalErrorRespectJSON("--to is required (target entity ID)")
		}
		if relType == "" {
			FatalErrorRespectJSON("--type is required (relationship type)")
		}

		// Validate confidence range
		if confidence < 0.0 || confidence > 1.0 {
			FatalErrorRespectJSON("--confidence must be between 0.0 and 1.0, got %.2f", confidence)
		}

		// Parse valid-from (default to now if not specified)
		var validFrom time.Time
		if validFromStr != "" {
		parsed, err := timeparsing.ParseRelativeTime(validFromStr, time.Now())
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-from: %v", err)
			}
			validFrom = parsed
		} else {
			validFrom = time.Now().UTC()
		}

		// Parse valid-until (optional)
		var validUntil *time.Time
		if validUntilStr != "" {
		parsed, err := timeparsing.ParseRelativeTime(validUntilStr, time.Now())
			if err != nil {
				FatalErrorRespectJSON("invalid --valid-until: %v", err)
			}
			validUntil = &parsed

			// Validate temporal window
			if validUntil.Before(validFrom) {
				FatalErrorRespectJSON("--valid-until must be after --valid-from")
			}
		}

		// Parse metadata (optional)
		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				FatalErrorRespectJSON("invalid --metadata JSON: %v", err)
			}
		}

		// Generate relationship ID
	// Use hash-based ID generation
	relID := idgen.GenerateHashID("rel", fromEntity+":"+relType+":"+toEntity, "", actor, time.Now(), 6, 0)
		// Get actor for attribution
		actor := getActorWithGit()

		// Create relationship
		rel := &types.Relationship{
			ID:               relID,
			SourceEntityID:   fromEntity,
			RelationshipType: relType,
			TargetEntityID:   toEntity,
			ValidFrom:        validFrom,
			ValidUntil:       validUntil,
			Confidence:       &confidence,
			Metadata:         metadata,
			CreatedAt:        time.Now().UTC(),
			CreatedBy:        actor,
		}

		// Store in database
		store := getStore()
		if err := store.CreateRelationship(ctx, rel); err != nil {
			FatalErrorRespectJSON("failed to create relationship: %v", err)
		}

		// Mark that we wrote data
		commandDidWrite.Store(true)

		// Output result
		if jsonOutput {
			data, err := json.MarshalIndent(rel, "", "  ")
			if err != nil {
				FatalErrorRespectJSON("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(data))
		} else {
			fmt.Printf("Created relationship %s: %s -[%s]-> %s\n",
				rel.ID, fromEntity, relType, toEntity)
			if validUntil != nil {
				fmt.Printf("Valid from %s to %s\n",
					validFrom.Format("2006-01-02 15:04"),
					validUntil.Format("2006-01-02 15:04"))
			} else {
				fmt.Printf("Valid from %s (indefinitely)\n",
					validFrom.Format("2006-01-02 15:04"))
			}
		}
	},
}

func init() {
	relationshipCreateCmd.Flags().String("from", "", "Source entity ID (required)")
	relationshipCreateCmd.Flags().String("to", "", "Target entity ID (required)")
	relationshipCreateCmd.Flags().String("type", "", "Relationship type (required)")
	relationshipCreateCmd.Flags().String("valid-from", "", "Start of validity window (default: now)")
	relationshipCreateCmd.Flags().String("valid-until", "", "End of validity window (optional)")
	relationshipCreateCmd.Flags().Float64("confidence", 1.0, "Confidence score 0.0-1.0 (default 1.0)")
	relationshipCreateCmd.Flags().String("metadata", "", "Additional metadata as JSON object")

	relationshipCmd.AddCommand(relationshipCreateCmd)
}
