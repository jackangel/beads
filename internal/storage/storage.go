// Package storage provides shared types for issue storage.
//
// The concrete storage implementation lives in the dolt sub-package.
// This package holds interface and value types that are referenced by
// both the dolt implementation and its consumers (cmd/bd, etc.).
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/steveyegge/beads/internal/types"
)

// ErrAlreadyClaimed is returned when attempting to claim an issue that is already
// claimed by another user. The error message contains the current assignee.
var ErrAlreadyClaimed = errors.New("issue already claimed")

// ErrNotFound is returned when a requested entity does not exist in the database.
var ErrNotFound = errors.New("not found")

// ErrNotInitialized is returned when the database has not been initialized
// (e.g., issue_prefix config is missing).
var ErrNotInitialized = errors.New("database not initialized")

// ErrPrefixMismatch is returned when an issue ID does not match the configured prefix.
var ErrPrefixMismatch = errors.New("prefix mismatch")

// EntityStore defines operations for entity management (knowledge graph nodes).
// Entities represent any trackable object in the system - not just issues, but also
// people, components, documents, or any domain concept that can have relationships.
type EntityStore interface {
	// CreateEntity creates a new entity in the system.
	// The entity ID must be unique and follow the configured ID format.
	CreateEntity(ctx context.Context, entity *types.Entity) error

	// GetEntity retrieves an entity by its ID.
	// Returns ErrNotFound if the entity does not exist.
	GetEntity(ctx context.Context, id string) (*types.Entity, error)

	// UpdateEntity updates an existing entity's fields.
	// Only non-nil fields in the entity parameter are updated.
	UpdateEntity(ctx context.Context, entity *types.Entity) error

	// DeleteEntity removes an entity from the system.
	// This operation may fail if the entity has dependencies that prevent deletion.
	DeleteEntity(ctx context.Context, id string) error

	// SearchEntities finds entities matching the provided filters.
	// Returns an empty slice if no entities match the criteria.
	SearchEntities(ctx context.Context, filters EntityFilters) ([]*types.Entity, error)
}

// EntityFilters defines search criteria for entities.
// All non-zero/non-empty fields are combined with AND logic.
type EntityFilters struct {
	// EntityType filters by the type of entity (e.g., "person", "component", "document").
	// Empty string matches all types.
	EntityType string

	// Name performs a partial case-insensitive match on the entity name.
	// Empty string matches all names.
	Name string

	// Metadata filters entities by matching metadata key-value pairs.
	// An entity matches if it contains all specified metadata entries.
	Metadata map[string]interface{}

	// CreatedBy filters entities by their creator.
	// Empty string matches all creators.
	CreatedBy string

	// Limit restricts the maximum number of results returned.
	// Zero or negative values return all matching results.
	Limit int

	// Offset skips the first N matching results for pagination.
	// Must be non-negative.
	Offset int
}

// RelationshipDirection specifies direction for relationship traversal.
type RelationshipDirection int

const (
	// RelationshipDirectionOutgoing traverses relationships where the entity is the source.
	RelationshipDirectionOutgoing RelationshipDirection = iota
	// RelationshipDirectionIncoming traverses relationships where the entity is the target.
	RelationshipDirectionIncoming
	// RelationshipDirectionBoth traverses relationships in both directions.
	RelationshipDirectionBoth
)

// RelationshipFilters defines search criteria for relationships.
// All non-zero/non-nil fields are combined with AND logic.
type RelationshipFilters struct {
	// SourceEntityID filters relationships by source entity.
	// Empty string matches all source entities.
	SourceEntityID string

	// TargetEntityID filters relationships by target entity.
	// Empty string matches all target entities.
	TargetEntityID string

	// RelationshipType filters by relationship type (e.g., "uses", "implements").
	// Empty string matches all types.
	RelationshipType string

	// ValidAt filters to relationships valid at a specific point in time.
	// Nil matches all relationships regardless of temporal validity.
	ValidAt *time.Time

	// ValidAtStart and ValidAtEnd define a time range for temporal filtering.
	// Relationships must be valid during at least part of this range.
	// Both must be set together to define a range; nil values are ignored.
	ValidAtStart *time.Time
	ValidAtEnd   *time.Time

	// Metadata filters relationships by matching metadata key-value pairs.
	// A relationship matches if it contains all specified metadata entries.
	Metadata map[string]interface{}

	// Limit restricts the maximum number of results returned.
	// Zero or negative values return all matching results.
	Limit int

	// Offset skips the first N matching results for pagination.
	// Must be non-negative.
	Offset int
}

// RelationshipStore defines operations for relationship management (knowledge graph edges with temporal validity).
// Relationships represent typed, directional connections between entities with support for temporal validity
// to track how connections evolve over time.
type RelationshipStore interface {
	// CreateRelationship creates a new relationship between entities.
	// The relationship ID must be unique. ValidFrom is required; ValidUntil is optional.
	CreateRelationship(ctx context.Context, rel *types.Relationship) error

	// GetRelationship retrieves a relationship by its ID.
	// Returns ErrNotFound if the relationship does not exist.
	GetRelationship(ctx context.Context, id string) (*types.Relationship, error)

	// UpdateRelationship updates an existing relationship's fields.
	// Only non-nil/non-zero fields in the relationship parameter are updated.
	UpdateRelationship(ctx context.Context, rel *types.Relationship) error

	// DeleteRelationship removes a relationship from the system.
	// This is a hard delete; for temporal validity, use UpdateRelationship to set ValidUntil instead.
	DeleteRelationship(ctx context.Context, id string) error

	// SearchRelationships finds relationships matching the provided filters.
	// Returns an empty slice if no relationships match the criteria.
	SearchRelationships(ctx context.Context, filters RelationshipFilters) ([]*types.Relationship, error)

	// GetRelationshipsWithTemporalFilter retrieves relationships for an entity with temporal and directional filtering.
	// The validAt parameter filters to relationships valid at the specified time.
	// The direction parameter controls whether to return outgoing, incoming, or both types of relationships.
	GetRelationshipsWithTemporalFilter(ctx context.Context, entityID string, validAt time.Time, direction RelationshipDirection) ([]*types.Relationship, error)
}

// EpisodeFilters defines search criteria for episodes.
// Episodes are immutable provenance logs, so all non-zero/non-nil fields are combined with AND logic.
type EpisodeFilters struct {
	// Source filters episodes by their data source (e.g., "github", "jira", "manual").
	// Empty string matches all sources.
	Source string

	// TimestampStart filters episodes from this timestamp (inclusive).
	// Nil matches all episodes regardless of start time.
	TimestampStart *time.Time

	// TimestampEnd filters episodes to this timestamp (inclusive).
	// Nil matches all episodes regardless of end time.
	TimestampEnd *time.Time

	// EntitiesExtracted filters by entity IDs that were extracted from the episode.
	// An episode matches if it extracted any of the specified entity IDs.
	// Empty slice matches all episodes.
	EntitiesExtracted []string

	// Limit restricts the maximum number of results returned.
	// Zero or negative values return all matching results.
	Limit int

	// Offset skips the first N matching results for pagination.
	// Must be non-negative.
	Offset int
}

// EpisodeStore defines operations for episode management (immutable provenance log).
// Episodes represent snapshots of ingested data and are never modified after creation.
// This interface intentionally omits Update and Delete operations to enforce immutability.
type EpisodeStore interface {
	// CreateEpisode creates a new episode in the system.
	// The episode ID must be unique and the episode is immutable after creation.
	CreateEpisode(ctx context.Context, episode *types.Episode) error

	// GetEpisode retrieves an episode by its ID.
	// Returns ErrNotFound if the episode does not exist.
	GetEpisode(ctx context.Context, id string) (*types.Episode, error)

	// SearchEpisodes finds episodes matching the provided filters.
	// Returns an empty slice if no episodes match the criteria.
	// Episodes are ordered by timestamp descending (newest first) by default.
	SearchEpisodes(ctx context.Context, filters EpisodeFilters) ([]*types.Episode, error)

	// Note: No UpdateEpisode or DeleteEpisode - episodes are immutable provenance records.
}

// OntologyStore defines operations for custom type registration and validation.
// This interface allows systems to define entity and relationship schemas with
// custom validation rules, constraints, and metadata requirements.
//
// Type schemas enable:
//   - Structured metadata with required/optional fields
//   - Validation rules and constraints
//   - Type hierarchies and inheritance
//   - Documentation and semantic meaning
type OntologyStore interface {
	// RegisterEntityType registers a new entity type schema in the system.
	// The schema defines the structure, required fields, and validation rules for entities of this type.
	// Returns an error if a schema with the same TypeName already exists.
	RegisterEntityType(ctx context.Context, schema *types.EntityTypeSchema) error

	// RegisterRelationshipType registers a new relationship type schema in the system.
	// The schema defines validation rules, allowed source/target entity types, and metadata requirements.
	// Returns an error if a schema with the same TypeName already exists.
	RegisterRelationshipType(ctx context.Context, schema *types.RelationshipTypeSchema) error

	// GetEntityTypes retrieves all registered entity type schemas.
	// Returns an empty slice if no entity types are registered.
	GetEntityTypes(ctx context.Context) ([]*types.EntityTypeSchema, error)

	// GetRelationshipTypes retrieves all registered relationship type schemas.
	// Returns an empty slice if no relationship types are registered.
	GetRelationshipTypes(ctx context.Context) ([]*types.RelationshipTypeSchema, error)

	// GetEntityTypeSchema retrieves the schema for a specific entity type by name.
	// Returns ErrNotFound if the type schema does not exist.
	GetEntityTypeSchema(ctx context.Context, typeName string) (*types.EntityTypeSchema, error)

	// GetRelationshipTypeSchema retrieves the schema for a specific relationship type by name.
	// Returns ErrNotFound if the type schema does not exist.
	GetRelationshipTypeSchema(ctx context.Context, typeName string) (*types.RelationshipTypeSchema, error)

	// ValidateEntityAgainstType validates an entity against a registered type schema.
	// Returns an error if the entity does not conform to the schema's requirements
	// (e.g., missing required fields, invalid field values, constraint violations).
	ValidateEntityAgainstType(ctx context.Context, entity *types.Entity, typeName string) error

	// ValidateRelationshipAgainstType validates a relationship against a registered type schema.
	// Returns an error if the relationship does not conform to the schema's requirements
	// (e.g., invalid source/target entity types, missing required metadata).
	ValidateRelationshipAgainstType(ctx context.Context, rel *types.Relationship, typeName string) error
}

// Storage is the interface satisfied by *dolt.DoltStore.
// Consumers depend on this interface rather than on the concrete type so that
// alternative implementations (mocks, proxies, etc.) can be substituted.
type Storage interface {
	// Entity operations (knowledge graph nodes)
	EntityStore

	// Relationship operations (knowledge graph edges with temporal validity)
	RelationshipStore

	// Episode operations (immutable provenance log)
	EpisodeStore

	// Ontology operations (custom type registration and validation)
	OntologyStore

	// Issue CRUD
	CreateIssue(ctx context.Context, issue *types.Issue, actor string) error
	CreateIssues(ctx context.Context, issues []*types.Issue, actor string) error
	GetIssue(ctx context.Context, id string) (*types.Issue, error)
	GetIssueByExternalRef(ctx context.Context, externalRef string) (*types.Issue, error)
	GetIssuesByIDs(ctx context.Context, ids []string) ([]*types.Issue, error)
	UpdateIssue(ctx context.Context, id string, updates map[string]interface{}, actor string) error
	CloseIssue(ctx context.Context, id string, reason string, actor string, session string) error
	DeleteIssue(ctx context.Context, id string) error
	SearchIssues(ctx context.Context, query string, filter types.IssueFilter) ([]*types.Issue, error)

	// Dependencies
	AddDependency(ctx context.Context, dep *types.Dependency, actor string) error
	RemoveDependency(ctx context.Context, issueID, dependsOnID string, actor string) error
	GetDependencies(ctx context.Context, issueID string) ([]*types.Issue, error)
	GetDependents(ctx context.Context, issueID string) ([]*types.Issue, error)
	GetDependenciesWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error)
	GetDependentsWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error)
	GetDependencyTree(ctx context.Context, issueID string, maxDepth int, showAllPaths bool, reverse bool) ([]*types.TreeNode, error)

	// Labels
	AddLabel(ctx context.Context, issueID, label, actor string) error
	RemoveLabel(ctx context.Context, issueID, label, actor string) error
	GetLabels(ctx context.Context, issueID string) ([]string, error)
	GetIssuesByLabel(ctx context.Context, label string) ([]*types.Issue, error)

	// Work queries
	GetReadyWork(ctx context.Context, filter types.WorkFilter) ([]*types.Issue, error)
	GetBlockedIssues(ctx context.Context, filter types.WorkFilter) ([]*types.BlockedIssue, error)
	GetEpicsEligibleForClosure(ctx context.Context) ([]*types.EpicStatus, error)

	// Comments and events
	AddIssueComment(ctx context.Context, issueID, author, text string) (*types.Comment, error)
	GetIssueComments(ctx context.Context, issueID string) ([]*types.Comment, error)
	GetEvents(ctx context.Context, issueID string, limit int) ([]*types.Event, error)
	GetAllEventsSince(ctx context.Context, since time.Time) ([]*types.Event, error)

	// Statistics
	GetStatistics(ctx context.Context) (*types.Statistics, error)

	// Configuration
	SetConfig(ctx context.Context, key, value string) error
	GetConfig(ctx context.Context, key string) (string, error)
	GetAllConfig(ctx context.Context) (map[string]string, error)

	// Transactions
	RunInTransaction(ctx context.Context, commitMsg string, fn func(tx Transaction) error) error

	// Lifecycle
	Close() error
}

// DoltStorage is the full interface for Dolt-backed stores, composing the core
// Storage interface with all capability sub-interfaces. Both DoltStore and
// EmbeddedDoltStore satisfy this interface.
type DoltStorage interface {
	Storage
	VersionControl
	HistoryViewer
	RemoteStore
	SyncStore
	FederationStore
	BulkIssueStore
	DependencyQueryStore
	AnnotationStore
	ConfigMetadataStore
	CompactionStore
	AdvancedQueryStore
}

// Transaction provides atomic multi-operation support within a single database transaction.
//
// The Transaction interface exposes a subset of storage methods that execute within
// a single database transaction. This enables atomic workflows where multiple operations
// must either all succeed or all fail (e.g., creating issues with dependencies and labels).
//
// # Transaction Semantics
//
//   - All operations within the transaction share the same database connection
//   - Changes are not visible to other connections until commit
//   - If any operation returns an error, the transaction is rolled back
//   - If the callback function panics, the transaction is rolled back
//   - On successful return from the callback, the transaction is committed
//
// # Example Usage
//
//	err := store.RunInTransaction(ctx, "bd: create parent and child", func(tx storage.Transaction) error {
//	    // Create parent issue
//	    if err := tx.CreateIssue(ctx, parentIssue, actor); err != nil {
//	        return err // Triggers rollback
//	    }
//	    // Create child issue
//	    if err := tx.CreateIssue(ctx, childIssue, actor); err != nil {
//	        return err // Triggers rollback
//	    }
//	    // Add dependency between them
//	    if err := tx.AddDependency(ctx, dep, actor); err != nil {
//	        return err // Triggers rollback
//	    }
//	    return nil // Triggers commit
//	})
type Transaction interface {
	// Issue operations
	CreateIssue(ctx context.Context, issue *types.Issue, actor string) error
	CreateIssues(ctx context.Context, issues []*types.Issue, actor string) error
	UpdateIssue(ctx context.Context, id string, updates map[string]interface{}, actor string) error
	CloseIssue(ctx context.Context, id string, reason string, actor string, session string) error
	DeleteIssue(ctx context.Context, id string) error
	GetIssue(ctx context.Context, id string) (*types.Issue, error)                                    // For read-your-writes within transaction
	SearchIssues(ctx context.Context, query string, filter types.IssueFilter) ([]*types.Issue, error) // For read-your-writes within transaction

	// Dependency operations
	AddDependency(ctx context.Context, dep *types.Dependency, actor string) error
	RemoveDependency(ctx context.Context, issueID, dependsOnID string, actor string) error
	GetDependencyRecords(ctx context.Context, issueID string) ([]*types.Dependency, error)

	// Label operations
	AddLabel(ctx context.Context, issueID, label, actor string) error
	RemoveLabel(ctx context.Context, issueID, label, actor string) error
	GetLabels(ctx context.Context, issueID string) ([]string, error)

	// Config operations (for atomic config + issue workflows)
	SetConfig(ctx context.Context, key, value string) error
	GetConfig(ctx context.Context, key string) (string, error)

	// Metadata operations (for internal state like import hashes)
	SetMetadata(ctx context.Context, key, value string) error
	GetMetadata(ctx context.Context, key string) (string, error)

	// Comment operations
	AddComment(ctx context.Context, issueID, actor, comment string) error
	ImportIssueComment(ctx context.Context, issueID, author, text string, createdAt time.Time) (*types.Comment, error)
	GetIssueComments(ctx context.Context, issueID string) ([]*types.Comment, error)
}
