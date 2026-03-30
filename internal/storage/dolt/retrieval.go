package dolt

import (
	"context"

	"github.com/steveyegge/beads/internal/retrieval"
	"github.com/steveyegge/beads/internal/storage"
)

// RetrieveMemory assembles relevant context from the knowledge graph.
// This method delegates to the retrieval package which handles:
// 1. Semantic search for initial entities (TextQuery)
// 2. Graph traversal from initial entities (MaxHops)
// 3. Temporal filtering on relationships (ValidAt)
// 4. Episode lookup for provenance
func (d *DoltStore) RetrieveMemory(ctx context.Context, query storage.MemoryQuery) (*storage.MemoryContext, error) {
	// Delegate to retrieval package
	return retrieval.RetrieveMemory(ctx, d, query)
}
