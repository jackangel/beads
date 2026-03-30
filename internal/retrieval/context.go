package retrieval

import (
	"context"
	"time"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

// RetrieveMemory assembles relevant context from knowledge graph.
// 1. Semantic search for initial entities (TextQuery)
// 2. Graph traversal from initial entities (MaxHops)
// 3. Temporal filtering on relationships (ValidAt)
// 4. Episode lookup for provenance
func RetrieveMemory(ctx context.Context, store storage.Storage, query storage.MemoryQuery) (*storage.MemoryContext, error) {
	validAt := query.ValidAt
	if validAt == nil {
		now := time.Now()
		validAt = &now
	}

	// Step 1: Semantic search for initial entities
	searchFilters := storage.EntityFilters{
		TextQuery: query.TextQuery,
		Limit:     query.TopK,
	}
	seedEntities, err := store.SearchEntities(ctx, searchFilters)
	if err != nil {
		return nil, err
	}

	// Step 2: Graph traversal (BFS from seed entities)
	entities := make(map[string]*types.Entity)
	relationships := make(map[string]*types.Relationship)

	for _, e := range seedEntities {
		entities[e.ID] = e
	}

	visited := make(map[string]bool)
	frontier := make([]string, 0)
	for _, e := range seedEntities {
		frontier = append(frontier, e.ID)
	}

	for hop := 0; hop < query.MaxHops && len(frontier) > 0; hop++ {
		nextFrontier := []string{}

		for _, entityID := range frontier {
			if visited[entityID] {
				continue
			}
			visited[entityID] = true

			// Get all relationships (outgoing and incoming)
			relFilters := storage.RelationshipFilters{
				ValidAt:       validAt,
				MinConfidence: &query.MinConfidence,
			}

			// Outgoing
			relFilters.SourceEntityID = entityID
			rels, _ := store.SearchRelationships(ctx, relFilters)
			for _, rel := range rels {
				relationships[rel.ID] = rel
				if !visited[rel.TargetEntityID] {
					nextFrontier = append(nextFrontier, rel.TargetEntityID)
				}
			}

			// Incoming
			relFilters.SourceEntityID = ""
			relFilters.TargetEntityID = entityID
			rels, _ = store.SearchRelationships(ctx, relFilters)
			for _, rel := range rels {
				relationships[rel.ID] = rel
				if !visited[rel.SourceEntityID] {
					nextFrontier = append(nextFrontier, rel.SourceEntityID)
				}
			}
		}

		// Fetch entities for next frontier
		for _, id := range nextFrontier {
			if entities[id] == nil {
				e, _ := store.GetEntity(ctx, id)
				if e != nil {
					entities[id] = e
				}
			}
		}

		frontier = nextFrontier
	}

	// Step 3: Lookup source episodes (entities_extracted includes entity IDs)
	// TBD: Add GetEpisodesByEntityID method or filter SearchEpisodes

	// Assemble result
	result := &storage.MemoryContext{
		Entities:        make([]*types.Entity, 0, len(entities)),
		Relationships:   make([]*types.Relationship, 0, len(relationships)),
		RelevanceScores: make(map[string]float64),
	}

	for _, e := range entities {
		result.Entities = append(result.Entities, e)
	}
	for _, rel := range relationships {
		result.Relationships = append(result.Relationships, rel)
	}

	// Relevance: seed entities score 1.0, others decay by hop distance
	for _, e := range seedEntities {
		result.RelevanceScores[e.ID] = 1.0
	}

	return result, nil
}
