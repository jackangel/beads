# Beads Codebase Research Bundle

*Generated: 2026-03-30 by EditMode_Researcher*
*Purpose: One-time context snapshot for 6-feature implementation plan*

---

## 1. Architecture Summary

Beads is a **Dolt-powered knowledge graph + issue tracker** CLI (`bd`). The project has two coexisting layers:

- **v7 (issues)**  legacy issue tracker with 14+ tables (`issues`, `dependencies`, `labels`, etc.)
- **v8 (entities)**  new knowledge graph layer with 5 tables (`entities`, `relationships`, `episodes`, `entity_types`, `relationship_types`)

### Entry Points
- **CLI**: `cmd/bd/main.go`, ~100+ Cobra commands, all support `--json`
- **Go Library**: `beads.go`  `Open()`, `OpenFromConfig()`, `FindDatabasePath()`
- **MCP Server**: `integrations/beads-mcp/`  Python FastMCP 3.1.1 wrapping `bd` CLI subprocess

### Storage Stack
```
cmd/bd    internal/storage/storage.go (interfaces)
                
       internal/storage/dolt/*.go  (DoltStore implementation)
                
         .beads/dolt/  (Dolt versioned MySQL-compatible DB)
```

**Key interfaces** (all in `internal/storage/storage.go`):
- `EntityStore`  CRUD for entities + `SearchEntities`
- `RelationshipStore`  CRUD for relationships + temporal filtering
- `EpisodeStore`  Create/Get/Search for immutable episodes
- `OntologyStore`  type schema registration/validation
- `Storage`  composes all sub-interfaces + legacy issue ops
- `DoltStorage`  `Storage` + VersionControl + HistoryViewer + RemoteStore + ...

### Hook System
`internal/hooks/hooks.go`  file-based hooks in `.beads/hooks/`:
- `on_create`, `on_update`, `on_close`  **issue events only**
- Async fire-and-forget, 10s timeout
- Issue JSON sent via stdin to hook script
- **No entity/episode hooks exist yet**

### Query Engine
`internal/query/`  expression parser for **issue filtering only**:
- `Lexer`  `Parser`  `Evaluator`
- Two modes: SQL-only (AND chains), predicate+SQL (OR/complex)
- Syntax: `status=open AND priority<2 AND updated>7d`
- `QueryResult{Filter types.IssueFilter, Predicate func(*types.Issue) bool}`

---

## 2. Feature 1  Entity Extraction Pipeline

### What Exists

**Episode ingestion** (`cmd/bd/episode_create.go`):
```go
// Reads raw data from file, stores as BLOB, accepts optional entity IDs
episode := &types.Episode{
    ID:                episodeID,  // idgen.GenerateHashID("ep", ...)
    Timestamp:         timestamp,
    Source:            source,     // e.g., "github", "jira", "manual"
    RawData:           rawData,    // raw file bytes
    EntitiesExtracted: entitiesExtracted,  // manually provided []string
}
store.CreateEpisode(ctx, episode)
```

**Episode type** (`internal/types/episode.go`):
```go
type Episode struct {
    ID                string                 `json:"id"`
    Timestamp         time.Time              `json:"timestamp"`
    Source            string                 `json:"source"`
    RawData           []byte                 `json:"raw_data"`         // BLOB storage
    EntitiesExtracted []string               `json:"entities_extracted,omitempty"` // entity IDs
    Metadata          map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt         time.Time              `json:"created_at"`
}
```

**Entity creation** (`cmd/bd/entity_create.go`):
```go
// Creates entity with: --entity-type, --name, --summary, --metadata, --created-by, --id
bd entity create --entity-type person --name "Alice" --summary "Senior engineer"
```

**External integrations** (`internal/github/`, `internal/gitlab/`, `internal/jira/`, `internal/linear/`):
- Pull/Push `bd` CLI operations via `internal/tracker/` framework
- `internal/tracker/`  plugin adapter with hooks: `GenerateID`, `TransformIssue`, `ShouldImport`, `ShouldPush`
- These sync **issues**, not entities/episodes

**Hooks** (`internal/hooks/hooks.go`):
```go
// Only issue events exist
const (
    EventCreate = "create"
    EventUpdate = "update"
    EventClose  = "close"
)
// Hook files: on_create, on_update, on_close
// Fired by: store write ops  hooks.Runner.Run(event, issue)
```

### What is Missing
1. **No LLM extraction pipeline**  `RawData` is stored but never processed
2. **No extraction trigger**  no event fires after `CreateEpisode`
3. **No entity/episode hooks**  hook system only fires on issue events
4. **No episode-entity linkage**  `EntitiesExtracted` must be manually populated
5. **No CLI command**  no `bd episode extract` or `bd episode process`

### Relevant Files for Implementation
- `internal/types/episode.go`  Episode struct (extend with `ProcessedAt *time.Time`)
- `cmd/bd/episode_create.go`  where to add `--extract` flag or a new `episode extract` subcommand
- `cmd/bd/find_duplicates.go`  AI call pattern using `anthropic-sdk-go` (reusable)
- `internal/hooks/hooks.go`  could add `EventEpisodeCreated` hook
- `internal/storage/storage.go`  `EpisodeStore` interface, `EntityStore.CreateEntity`

### `find_duplicates.go` AI Call Pattern (reusable)
```go
// Uses github.com/anthropics/anthropic-sdk-go
import "github.com/anthropics/anthropic-sdk-go"
import "github.com/anthropics/anthropic-sdk-go/option"

client := anthropic.NewClient(option.WithAPIKey(apiKey))
msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
    Model:     anthropic.F(model),
    MaxTokens: anthropic.F(int64(2048)),
    Messages:  anthropic.F([]anthropic.MessageParam{...}),
})
// Parse response: msg.Content[0].AsText().Text
```

---

## 3. Feature 2  Semantic/Vector Search

### Current `SearchEntities` Signature
```go
// internal/storage/storage.go
type EntityFilters struct {
    EntityType string
    Name       string                 // SQL LIKE "%name%"
    Metadata   map[string]interface{}
    CreatedBy  string
    Limit      int
    Offset     int
}

SearchEntities(ctx context.Context, filters EntityFilters) ([]*types.Entity, error)
```

### Current SQL Implementation (`internal/storage/dolt/entities.go`)
```go
// Name filter uses LIKE:
whereClauses = append(whereClauses, "name LIKE ?")
args = append(args, "%"+filters.Name+"%")

// Full query:
// SELECT id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by
// FROM entities [WHERE ...] ORDER BY created_at DESC [LIMIT N] [OFFSET N]
```

### What is Missing
1. **No embedding/vector columns** in entities or relationships tables
2. **No vector store** dependency in `go.mod`
3. **No embedding generation** anywhere in the codebase
4. **No similarity search** for entities (only Jaccard/cosine for issues in `find_duplicates.go`)

### `go.mod`  Vector/ML-Relevant Dependencies
```
github.com/anthropics/anthropic-sdk-go v1.26.0   can generate embeddings via Messages API
// NO pgvector, NO chromem-go, NO qdrant, NO sqlite-vss
// NO OpenAI SDK, NO Cohere
```

### Gaps
- **`EntityFilters` has no `SemanticQuery string` or `Embedding []float32` field**
- **`SearchEntities` has no vector path**  would need either:
  - New field in `EntityFilters` + in-memory cosine similarity post-SQL-fetch
  - Separate `SearchEntitiesSemantic(ctx, query string) ([]*types.Entity, error)` on `EntityStore`
  - External vector store (pgvector, sqlite-vec, Qdrant) integrated with Dolt
- **Anthropic SDK available** but no embeddings endpoint  need Claude via few-shot scoring or use a separate model
- **Cheapest viable approach**: fetch all entities by type, compute cosine similarity in-memory using entity name+summary tokens (same approach as `find_duplicates.go` Jaccard)

### `SearchEpisodes`  No Semantic Either
```go
type EpisodeFilters struct {
    Source            string
    TimestampStart    *time.Time
    TimestampEnd      *time.Time
    EntitiesExtracted []string  // matches ANY of these entity IDs
    Limit             int
    Offset            int
}
// No text search in RawData
```

---

## 4. Feature 3  Entity Deduplication/Resolution

### Current Entity ID Generation (`cmd/bd/entity_create.go`)
```go
entityID = idgen.GenerateHashID("ent", entityType+":"+name, summary, actor, time.Now(), 6, 0)
```

`idgen.GenerateHashID` (`internal/idgen/hash.go`):
```go
func GenerateHashID(prefix, title, description, creator string, timestamp time.Time, length, nonce int) string {
    content := fmt.Sprintf("%s|%s|%s|%d|%d", title, description, creator, timestamp.UnixNano(), nonce)
    hash := sha256.Sum256([]byte(content))
    shortHash := EncodeBase36(hash[:numBytes], length)  // base36, 4 bytes  6 chars
    return fmt.Sprintf("%s-%s", prefix, shortHash)      // e.g., "ent-a3f8e9"
}
```

**Key observation**: `timestamp.UnixNano()` is included  two entities with same name/type created at different times get **different IDs**. No deduplication by content hash.

### Issue Duplicate Detection (`cmd/bd/find_duplicates.go`)
This is the only existing deduplication logic in the codebase. Key algorithms:

```go
// tokenize: lowercase, remove punct, single-char words excluded
func tokenize(text string) map[string]int

// jaccardSimilarity: intersection/union of token counts
func jaccardSimilarity(a, b map[string]int) float64

// cosineSimilarity: dot product / (|a| * |b|)
func cosineSimilarity(a, b map[string]int) float64

// findMechanicalDuplicates: O(n^2) naive comparison, threshold 0.5 default
// findAIDuplicates: mechanical pre-filter  LLM classification
```

`duplicatePair` struct:
```go
type duplicatePair struct {
    IssueA     *types.Issue `json:"issue_a"`
    IssueB     *types.Issue `json:"issue_b"`
    Similarity float64      `json:"similarity"`
    Method     string       `json:"method"`
    Reason     string       `json:"reason,omitempty"`
}
```

### What is Missing for Entity Deduplication
1. **No entity merge operation**  no `MergeEntities(ctx, sourceID, targetID string)` in storage interface
2. **No canonical name normalization**  different capitalizations create separate entities
3. **No duplicate detection for entities**  `find_duplicates.go` is issue-only
4. **`UpdateEntity` replaces fields entirely**  no merge/append of summaries
5. **Relationships don't update source/target** when an entity is merged/deleted (cascade delete only)
6. **No alias/also-known-as system**

### `UpdateEntity` Partial-Update Pattern (for merge reference)
```go
// internal/storage/dolt/entities.go
// Dynamic SET clauses  only non-zero fields are updated
setClauses := []string{"updated_at = ?"}
if entity.EntityType != "" { ... }
if entity.Name != "" { ... }
if entity.Summary != "" { ... }
if entity.Metadata != nil { ... }
```

---

## 5. Feature 4  Relationship Confidence/Weight

### Current `Relationship` Struct (`internal/types/relationship.go`)
```go
type Relationship struct {
    ID               string                 `json:"id"`
    SourceEntityID   string                 `json:"source_entity_id"`
    RelationshipType string                 `json:"relationship_type"`
    TargetEntityID   string                 `json:"target_entity_id"`
    ValidFrom        time.Time              `json:"valid_from"`
    ValidUntil       *time.Time             `json:"valid_until,omitempty"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt        time.Time              `json:"created_at"`
    CreatedBy        string                 `json:"created_by,omitempty"`
}

func (r *Relationship) IsValidAt(t time.Time) bool { ... }
```

**No `Confidence float64` or `Weight float64` field.**

### Current Schema (`internal/storage/dolt/schema_v8.sql`)
```sql
CREATE TABLE IF NOT EXISTS relationships (
    id                VARCHAR(255) PRIMARY KEY,
    source_entity_id  VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(255) NOT NULL,
    target_entity_id  VARCHAR(255) NOT NULL,
    valid_from        DATETIME NOT NULL,
    valid_until       DATETIME NULL,
    metadata          JSON DEFAULT (JSON_OBJECT()),   -- only current extension point
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by        VARCHAR(255) NOT NULL,
    -- FK cascade delete on both entity refs
);
```

### Metadata as Escape Hatch
Confidence could be stored in `metadata`:
```json
{"confidence": 0.87, "source": "llm-extraction", "extraction_model": "claude-3-5-sonnet"}
```

But this has costs:
- Not queryable via SQL `WHERE confidence > 0.5`
- No type validation
- No SQL index in Dolt (JSON `->` operator but not indexed by value)

### What is Needed for First-Class Confidence
1. **Add to Relationship struct**: `Confidence *float64 \`json:"confidence,omitempty"\``
2. **Migrate schema**: `ALTER TABLE relationships ADD COLUMN confidence FLOAT NULL`
3. **Update `SearchRelationships` filters**: add `MinConfidence *float64` to `RelationshipFilters`
4. **Update `CreateRelationship`/`UpdateRelationship`** in `dolt/relationships.go`
5. **Update `relationship_create.go`**: add `--confidence` flag

### `RelationshipFilters` Current State
```go
type RelationshipFilters struct {
    SourceEntityID   string
    TargetEntityID   string
    RelationshipType string
    ValidAt          *time.Time
    ValidAtStart     *time.Time
    ValidAtEnd       *time.Time
    Metadata         map[string]interface{}
    Limit            int
    Offset           int
}
// No MinConfidence, no MaxConfidence
```

---

## 6. Feature 5  Memory Retrieval Interface

### Current Search APIs

**Issue search** (`store.SearchIssues`):
```go
SearchIssues(ctx context.Context, query string, filter types.IssueFilter) ([]*types.Issue, error)
// query: title text search
// filter: IssueFilter{Status, Priority, Labels, Assignee, Dates, ...}
```

**Entity search** (`store.SearchEntities`):
```go
SearchEntities(ctx context.Context, filters EntityFilters) ([]*types.Entity, error)
// Filters: EntityType, Name (LIKE), CreatedBy, Metadata (map), Limit, Offset
// No full-text, no semantic, no cross-table join
```

**Episode search** (`store.SearchEpisodes`):
```go
SearchEpisodes(ctx context.Context, filters EpisodeFilters) ([]*types.Episode, error)
// Filters: Source, TimestampStart, TimestampEnd, EntitiesExtracted, Limit, Offset
// No text search in RawData
```

**Relationship search** (`store.SearchRelationships` + temporal):
```go
SearchRelationships(ctx context.Context, filters RelationshipFilters) ([]*types.Relationship, error)
GetRelationshipsWithTemporalFilter(ctx context.Context, entityID string, validAt time.Time, direction RelationshipDirection) ([]*types.Relationship, error)
// direction: RelationshipDirectionOutgoing|Incoming|Both
```

### Query Engine  Issue-Only (`internal/query/`)
```go
// Evaluator converts DSL  IssueFilter + optional predicate
e := query.NewEvaluator(time.Now())
result, err := e.Evaluate(node)  // node from parser
// result.Filter: types.IssueFilter
// result.Predicate: func(*types.Issue) bool (for OR/complex)
// result.RequiresPredicate: bool
```

The query engine is **completely issue-centric**  it knows fields like `status`, `priority`, `type`, `assignee`, `label`, `created`, `updated`. It cannot filter entities.

### `bd ready` Composition Pattern
```go
// Typical ready-work composition:
store.GetReadyWork(ctx, types.WorkFilter{
    Priority: priorityFilter,
    Assignee: assigneeFilter,
    Labels:   labelsFilter,
    Limit:    limit,
})
// WorkFilter is separate from IssueFilter  issue-centric
```

### What is Missing
1. **No `RetrieveContext(ctx, query string) (*MemoryContext, error)`**  no cross-table context assembly
2. **No graph traversal for context**  getting neighbors of an entity requires multiple manual `SearchRelationships` calls
3. **No relevance scoring**  no BM25, no TF-IDF, no cosine for entities
4. **No temporal context window**  "what was true as of 2026-01-01?"

### Graph Command Inventory (`cmd/bd/graph*.go`)
```
graph.go            parent command (bd graph)
graph_explore.go    bd graph explore (visual ASCII)
graph_export.go     bd graph export (GraphML, DOT formats)
graph_traverse.go   bd graph traverse (path finding)
graph_visual.go     bd graph visual (force-directed layout)
graph_visualize.go  aliases/helpers
```
These are display utilities, not retrieval search functions.

### `cmd/bd/memory.go`  Existing Memory System (NOT graph retrieval)
```go
// memory.go implements simple key-value memory via config table:
// bd remember "insight text" [--key slug]   store.SetConfig(ctx, "kv.memory."+key, insight)
// bd memories [search]                      store.GetAllConfig(ctx), filter "kv.memory.*"
```
This is a **text-note KV store**, completely separate from entity/relationship graph.

### Proposed Interface
```go
// New storage method or separate retrieval package:
type ContextOptions struct {
    MaxEntities    int
    MaxRelationships int
    ValidAt        *time.Time
    TraversalDepth int
}
type MemoryContext struct {
    Entities      []*types.Entity
    Relationships []*types.Relationship
    Episodes      []*types.Episode
    RelevanceScores map[string]float64
}
RetrieveContext(ctx context.Context, query string, opts ContextOptions) (*MemoryContext, error)
```

---

## 7. Feature 6  MCP Server Update

### Full Tool Inventory (`integrations/beads-mcp/src/beads_mcp/tools.py`)

All current tools are **issue-centric** (18 tools, 0 entity/relationship/episode tools):

| Tool Function | CLI Equivalent | Purpose |
|--------------|---------------|---------|
| `beads_ready_work(limit, priority, assignee, labels, labels_any, unassigned, sort_policy, parent)` | `bd ready` | Unblocked issues |
| `beads_list_issues(status, priority, issue_type, assignee, labels, labels_any, query, unassigned, limit)` | `bd list` | Filter/list issues |
| `beads_show_issue(issue_id)` | `bd show <id>` | Issue details |
| `beads_create_issue(title, description, design, acceptance, external_ref, priority, issue_type, assignee, labels, id, deps)` | `bd create` | Create issue |
| `beads_update_issue(issue_id, status, priority, assignee, title, description, design, acceptance_criteria, notes, external_ref)` | `bd update` | Update issue |
| `beads_claim_issue(issue_id)` | `bd update --claim` | Atomic claim |
| `beads_close_issue(issue_id, reason)` | `bd close` | Close issue |
| `beads_reopen_issue(issue_ids, reason)` | `bd reopen` | Reopen issues |
| `beads_add_dependency(issue_id, depends_on_id, dep_type)` | `bd dep add` | Add dep |
| `beads_quickstart()` | `bd quickstart` | Guide text |
| `beads_stats()` | `bd stats` | Issue statistics |
| `beads_blocked(parent)` | `bd blocked` | Blocked issues |
| `beads_inspect_migration()` | `bd doctor inspect-migration` | DB migration state |
| `beads_get_schema_info()` | internal | Schema inspection |
| `beads_repair_deps(fix)` | `bd doctor repair-deps` | Fix orphan deps |
| `beads_detect_pollution(clean)` | `bd doctor detect-pollution` | Test pollution |
| `beads_validate(checks, fix_all)` | `bd doctor validate` | Health checks |
| `beads_init(prefix)` | `bd init` | Init database |

### `bd_client.py`  Abstract Methods on `BdClientBase`
```python
class BdClientBase(ABC):
    async def ready(self, params: Optional[ReadyWorkParams]) -> List[Issue]
    async def list_issues(self, params: Optional[ListIssuesParams]) -> List[Issue]
    async def show(self, params: ShowIssueParams) -> Issue
    async def create(self, params: CreateIssueParams) -> Issue
    async def update(self, params: UpdateIssueParams) -> Issue
    async def claim(self, params: ClaimIssueParams) -> Issue
    async def close(self, params: CloseIssueParams) -> List[Issue]
    async def reopen(self, params: ReopenIssueParams) -> List[Issue]
    async def add_dependency(self, params: AddDependencyParams) -> None
    async def quickstart(self) -> str
    async def stats(self) -> Stats
    async def blocked(self, params: Optional[BlockedParams]) -> List[BlockedIssue]
    async def init(self, params: Optional[InitParams]) -> str
    async def inspect_migration(self) -> dict[str, Any]
    async def get_schema_info(self) -> dict[str, Any]
    async def repair_deps(self, fix: bool) -> dict[str, Any]
    async def detect_pollution(self, clean: bool) -> dict[str, Any]
    async def validate(self, checks, fix_all) -> dict[str, Any]
    # + abstract: ping (optional health check)
```

### CLI Subprocess Pattern (from `bd_client.py` concrete implementation)
```python
result = subprocess.run(
    ["bd", "--json", "entity", "create", "--entity-type", entity_type, "--name", name],
    cwd=working_dir,
    capture_output=True, text=True, check=False,
    shell=sys.platform == "win32",
    stdin=subprocess.DEVNULL,  # critical: never inherit MCP stdin
)
data = json.loads(result.stdout)
```

### Pydantic Models (`models.py`)  Complete Inventory

```python
# Type aliases
IssueStatus = str
IssueType = str
DependencyType = Literal["blocks", "related", "parent-child", "discovered-from"]
OperationAction = Literal["created", "updated", "claimed", "closed", "reopened"]

# Lightweight/compact models
class IssueMinimal(BaseModel): id, title, status, priority, issue_type, assignee, labels, dependency_count, dependent_count
class CompactedResult(BaseModel): compacted, total_count, preview: list[IssueMinimal], preview_count, hint
class BriefIssue(BaseModel): id, title, status, priority
class BriefDep(BaseModel): id, title, status, priority, dependency_type
class OperationResult(BaseModel): id, action, message

# Full models
class IssueBase(BaseModel): id, title, description, design, acceptance_criteria, notes, external_ref, status, priority, issue_type, created_at, updated_at, closed_at, assignee, labels, dependency_count, dependent_count
class LinkedIssue(IssueBase): dependency_type
class Issue(IssueBase): dependencies: list[LinkedIssue], dependents: list[LinkedIssue]
class Dependency(BaseModel): from_id, to_id, dep_type

# Params models
class CreateIssueParams(BaseModel): title, description, design, acceptance, external_ref, priority, issue_type, assignee, labels, id, deps
class UpdateIssueParams(BaseModel): issue_id, status, priority, assignee, title, description, design, acceptance_criteria, notes, external_ref
class ClaimIssueParams(BaseModel): issue_id
class CloseIssueParams(BaseModel): issue_id, reason
class ReopenIssueParams(BaseModel): issue_ids, reason
class AddDependencyParams(BaseModel): issue_id, depends_on_id, dep_type
class ReadyWorkParams(BaseModel): limit, priority, assignee, labels, labels_any, unassigned, sort_policy, parent_id
class BlockedParams(BaseModel): parent_id
class ListIssuesParams(BaseModel): status, priority, issue_type, assignee, labels, labels_any, query, unassigned, limit
class ShowIssueParams(BaseModel): issue_id

# Stats models
class StatsSummary(BaseModel): total_issues, open_issues, in_progress_issues, closed_issues, blocked_issues, deferred_issues, ready_issues, tombstone_issues, pinned_issues, epics_eligible_for_closure, average_lead_time_hours
class RecentActivity(BaseModel): hours_tracked, commit_count, issues_created, issues_closed, issues_updated, issues_reopened, total_changes
class Stats(BaseModel): summary: StatsSummary, recent_activity: RecentActivity
class BlockedIssue(Issue): blocked_by_count, blocked_by: list[str]

# Init
class InitParams(BaseModel): prefix
class InitResult(BaseModel): database, prefix, message
```

**Missing models** to add for entity/relationship/episode support:
```python
class Entity(BaseModel): id, entity_type, name, summary, metadata, created_at, updated_at, created_by, updated_by
class Relationship(BaseModel): id, source_entity_id, relationship_type, target_entity_id, valid_from, valid_until, metadata, created_at, created_by
class Episode(BaseModel): id, timestamp, source, raw_data_size, entities_extracted, metadata, created_at
class CreateEntityParams(BaseModel): entity_type, name, summary, metadata, created_by, id
class EntitySearchParams(BaseModel): entity_type, name, limit, offset, created_by
class CreateRelationshipParams(BaseModel): from_entity, to_entity, relationship_type, valid_from, valid_until, metadata
class CreateEpisodeParams(BaseModel): source, file_path, timestamp, entities_extracted
```

### Test Pattern (`tests/test_tools.py`)
```python
@pytest.fixture(autouse=True)
def reset_connection_pool():
    from beads_mcp import tools
    tools._connection_pool.clear()
    yield
    tools._connection_pool.clear()

@pytest.mark.asyncio
async def test_beads_ready_work(sample_issue):
    mock_client = AsyncMock()
    mock_client.ready = AsyncMock(return_value=[sample_issue])
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        issues = await beads_ready_work(limit=10, priority=1)
    assert len(issues) == 1
    mock_client.ready.assert_called_once()
```

**Pattern**: mock `_get_client`, return `AsyncMock` with specific method returns, verify call count.

### `conftest.py` Safety Checks
- Sets `BEADS_TEST_MODE=1` on configure
- Blocks tests if `BEADS_DB` points to production `.beads/`
- Blocks tests if `BEADS_WORKING_DIR` is project root

---

## 8. Dependencies

### Go (`go.mod`)
```
module github.com/steveyegge/beads
go 1.25.8

# AI (available for extraction pipeline)
github.com/anthropics/anthropic-sdk-go v1.26.0

# Database
github.com/dolthub/driver v0.2.1-0.20260314000741-...  (embedded Dolt)
github.com/dolthub/dolt/go v0.40.5-0.20260313234613-...
github.com/dolthub/go-mysql-server v0.20.1-0.20260313230549-...
github.com/go-sql-driver/mysql v1.9.3

# CLI framework
github.com/spf13/cobra v1.10.2
github.com/spf13/viper v1.21.0

# Schema validation
github.com/xeipuuv/gojsonschema v1.2.0

# Observability
go.opentelemetry.io/otel v1.42.0

# Utilities
github.com/google/uuid v1.6.0
github.com/cenkalti/backoff/v4 v4.3.0
github.com/stretchr/testify v1.11.1
```

**Not present (would need adding for vector search)**:
- No `github.com/philippgille/chromem-go`
- No `github.com/qdrant/go-client`
- No OpenAI Go SDK
- No `golang.org/x/exp`

### Python (`integrations/beads-mcp/pyproject.toml`)
```toml
name = "beads-mcp"
version = "0.61.0"
requires-python = ">=3.10"

dependencies = [
    "fastmcp==3.1.1",
    "pydantic==2.12.5",
    "pydantic-settings==2.13.1",
]
```

**Not present (needed for in-MCP LLM calls)**:
- No `anthropic`
- No `openai`
- No `numpy` (cosine similarity)
- No `sentence-transformers`

---

## 9. Test Patterns

### Go Storage Tests (`internal/storage/dolt/dolt_test.go`)
```go
// Build tag: requires CGO + dolt server
// Semaphore to limit concurrency: var testSem = make(chan struct{}, 2)

func setupTestStore(t *testing.T) (*DoltStore, func()) {
    t.Helper()
    skipIfNoDolt(t)   // skips if dolt binary not found or test server not running
    t.Parallel()       // MUST call t.Parallel() BEFORE acquireTestSlot()
    acquireTestSlot()  // blocks until slot available
    t.Cleanup(releaseTestSlot)
    
    tmpDir, _ := os.MkdirTemp("", "dolt-test-*")
    // Each test gets own branch in shared test DB (COW snapshot)
    // uniqueTestDBName()  "testdb_" + random hex
}
```

**No `entities_test.go` or `relationships_test.go` exist yet**  these are gaps to fill.

### Go CLI Tests (`cmd/bd/cli_fast_test.go`)
```go
//go:build cgo && integration

// Template DB: initialized once, copied per test (avoids re-running bd-init)
var (
    templateDBDir  string
    templateDBOnce sync.Once
)

func setupCLITestDB(t *testing.T) string {
    // copies templateDBDir  t.TempDir()
    // template uses t.TempDir() equivalent via os.MkdirTemp
}

// In-process invocation pattern:
func TestEntityCreate(t *testing.T) {
    dbPath := setupCLITestDB(t)
    inProcessMutex.Lock()
    defer inProcessMutex.Unlock()
    os.Chdir(dbPath)
    
    var buf bytes.Buffer
    rootCmd.SetOut(&buf)
    rootCmd.SetArgs([]string{"--json", "entity", "create",
        "--entity-type", "person", "--name", "Alice"})
    require.NoError(t, rootCmd.Execute())
    
    var result map[string]interface{}
    require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
    assert.Equal(t, "person", result["entity_type"])
}
```

### Python MCP Tests (`integrations/beads-mcp/tests/`)
```python
# Standard mock pattern for all tool tests:
@pytest.mark.asyncio
async def test_some_entity_tool():
    entity = Entity(id="ent-abc", entity_type="person", name="Alice", ...)
    mock_client = AsyncMock()
    mock_client.create_entity = AsyncMock(return_value=entity)
    
    with patch("beads_mcp.tools._get_client", return_value=mock_client):
        result = await beads_create_entity(entity_type="person", name="Alice", summary="...")
    
    assert result.id == "ent-abc"
    mock_client.create_entity.assert_called_once()
```

---

## 10. Risk Areas

### HIGH: Storage Interface (`internal/storage/storage.go`)
- `Storage` interface is satisfied by `DoltStore` and used by all ~100 commands
- Adding `Confidence` to `Relationship` = struct change + schema migration + CRUD updates + filter updates
- Adding `SemanticQuery` to `EntityFilters` = struct change + dolt/entities.go implementation
- Any interface addition requires updating `DoltStorage` composed interface too

### HIGH: Schema Migrations (`internal/storage/dolt/schema_v8.sql`)
- Adding columns to `relationships` table requires `ALTER TABLE` migration
- Dolt schema migrations must be idempotent (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`)
- Migration framework: `cmd/bd/migrate.go` + `migrate_to_v8.go`
- All storage tests that init the schema need refreshing

### MEDIUM: Episode Processing  No Hook Point
- `CreateEpisode` in `dolt/episodes.go` does not call any hooks
- Hook runner (`internal/hooks/hooks.go`) only accepts `*types.Issue`  needs extension or separate runner
- Need to choose: post-create hook, new `episode extract` command, or `--extract` flag on create

### MEDIUM: MCP `BdClientBase` (`bd_client.py`)
- Abstract base class  adding entity methods requires updating: ABC + concrete `CliClient` + `tools.py` + `models.py` + test mocks
- Currently 0 entity methods  adding 6+ methods is significant surface area

### MEDIUM: `Relationship` Struct (`internal/types/relationship.go`)
- Used in: `relationship_create.go`, `relationship_update.go`, `relationship_show.go`, `relationship_list.go`, `dolt/relationships.go`, `storage.go` interface
- Adding `Confidence *float64` is backward-compatible (omitempty JSON tag)
- All consumers do field-by-field access, not struct spread

### LOW: Entity CRUD (`internal/storage/dolt/entities.go`)
- Self-contained, well-structured
- No tests exist yet (no `entities_test.go`)  risk is writing tests, not breaking existing ones

### LOW: ID Generation (`internal/idgen/hash.go`)
- `GenerateHashID` includes `timestamp.UnixNano()`  same name/type at different times  different IDs
- Entity dedup must normalize inputs (lowercase, trim) before ID comparison or use separate lookup by name

### LOW: Similarity Algorithm Reuse (`cmd/bd/find_duplicates.go`)
- `tokenize`, `jaccardSimilarity`, `cosineSimilarity` are unexported functions in `package main`
- For entity dedup: extract to `internal/utils/similarity.go` or `internal/dedup/` package

---

## Appendix: Implementation File Map

| Feature | Key Files to Modify | Key Files to Create |
|---------|--------------------|--------------------|
| 1. Entity Extraction | `cmd/bd/episode_create.go`, `internal/hooks/hooks.go` | `cmd/bd/episode_extract.go`, `internal/extraction/llm.go` |
| 2. Semantic Search | `internal/storage/storage.go` (EntityFilters), `internal/storage/dolt/entities.go` | `internal/search/semantic.go` |
| 3. Entity Dedup | `internal/storage/storage.go` (add MergeEntities), `internal/storage/dolt/entities.go` | `cmd/bd/entity_deduplicate.go`, `internal/dedup/entity.go` |
| 4. Rel Confidence | `internal/types/relationship.go`, `internal/storage/storage.go` (RelationshipFilters), `internal/storage/dolt/relationships.go`, `internal/storage/dolt/schema_v8.sql` | migration SQL snippet |
| 5. Memory Retrieval | `internal/storage/storage.go` (new interface method) | `cmd/bd/memory_retrieve.go`, `internal/retrieval/context.go` |
| 6. MCP Update | `integrations/beads-mcp/src/beads_mcp/models.py`, `integrations/beads-mcp/src/beads_mcp/bd_client.py`, `integrations/beads-mcp/src/beads_mcp/tools.py` | `integrations/beads-mcp/tests/test_entity_tools.py` |
