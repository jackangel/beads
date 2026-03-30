package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
)

const storageScopeName = "github.com/steveyegge/beads/storage"

// InstrumentedStorage wraps storage.Storage with OTel tracing and metrics.
// Every method gets a span and is counted in bd.storage.* metrics.
// Use WrapStorage to create one; it returns the original store unchanged when
// telemetry is disabled.
type InstrumentedStorage struct {
	inner      storage.Storage
	tracer     trace.Tracer
	ops        metric.Int64Counter
	dur        metric.Float64Histogram
	errs       metric.Int64Counter
	issueGauge metric.Int64Gauge
}

// WrapStorage returns s decorated with OTel instrumentation.
// When telemetry is disabled, s is returned as-is with zero overhead.
func WrapStorage(s storage.Storage) storage.Storage {
	if !Enabled() {
		return s
	}
	m := Meter(storageScopeName)
	ops, _ := m.Int64Counter("bd.storage.operations",
		metric.WithDescription("Total storage operations executed"),
	)
	dur, _ := m.Float64Histogram("bd.storage.operation.duration",
		metric.WithDescription("Storage operation duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	errs, _ := m.Int64Counter("bd.storage.errors",
		metric.WithDescription("Total storage operation errors"),
	)
	issueGauge, _ := m.Int64Gauge("bd.issue.count",
		metric.WithDescription("Current number of issues by status (snapshot from GetStatistics)"),
	)
	return &InstrumentedStorage{
		inner:      s,
		tracer:     Tracer(storageScopeName),
		ops:        ops,
		dur:        dur,
		errs:       errs,
		issueGauge: issueGauge,
	}
}

// op starts a span and records a metric for the named storage operation.
func (s *InstrumentedStorage) op(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span, time.Time) {
	all := append([]attribute.KeyValue{attribute.String("db.operation", name)}, attrs...)
	ctx, span := s.tracer.Start(ctx, "storage."+name,
		trace.WithAttributes(all...),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	s.ops.Add(ctx, 1, metric.WithAttributes(all...))
	return ctx, span, time.Now()
}

// done ends the span, records duration and optional error.
func (s *InstrumentedStorage) done(ctx context.Context, span trace.Span, start time.Time, err error, attrs ...attribute.KeyValue) {
	ms := float64(time.Since(start).Milliseconds())
	s.dur.Record(ctx, ms, metric.WithAttributes(attrs...))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.errs.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	span.End()
}

// ── Issue CRUD ──────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) CreateIssue(ctx context.Context, issue *types.Issue, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.actor", actor),
		attribute.String("bd.issue.type", string(issue.IssueType)),
	}
	ctx, span, t := s.op(ctx, "CreateIssue", attrs...)
	err := s.inner.CreateIssue(ctx, issue, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) CreateIssues(ctx context.Context, issues []*types.Issue, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.actor", actor),
		attribute.Int("bd.issue.count", len(issues)),
	}
	ctx, span, t := s.op(ctx, "CreateIssues", attrs...)
	err := s.inner.CreateIssues(ctx, issues, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetIssue(ctx context.Context, id string) (*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", id)}
	ctx, span, t := s.op(ctx, "GetIssue", attrs...)
	v, err := s.inner.GetIssue(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetIssueByExternalRef(ctx context.Context, externalRef string) (*types.Issue, error) {
	ctx, span, t := s.op(ctx, "GetIssueByExternalRef")
	v, err := s.inner.GetIssueByExternalRef(ctx, externalRef)
	s.done(ctx, span, t, err)
	return v, err
}

func (s *InstrumentedStorage) GetIssuesByIDs(ctx context.Context, ids []string) ([]*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.Int("bd.issue.count", len(ids))}
	ctx, span, t := s.op(ctx, "GetIssuesByIDs", attrs...)
	v, err := s.inner.GetIssuesByIDs(ctx, ids)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", id),
		attribute.String("bd.actor", actor),
		attribute.Int("bd.update.count", len(updates)),
	}
	ctx, span, t := s.op(ctx, "UpdateIssue", attrs...)
	err := s.inner.UpdateIssue(ctx, id, updates, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) CloseIssue(ctx context.Context, id string, reason string, actor string, session string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", id),
		attribute.String("bd.actor", actor),
	}
	ctx, span, t := s.op(ctx, "CloseIssue", attrs...)
	err := s.inner.CloseIssue(ctx, id, reason, actor, session)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) DeleteIssue(ctx context.Context, id string) error {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", id)}
	ctx, span, t := s.op(ctx, "DeleteIssue", attrs...)
	err := s.inner.DeleteIssue(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) SearchIssues(ctx context.Context, query string, filter types.IssueFilter) ([]*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.query", query)}
	ctx, span, t := s.op(ctx, "SearchIssues", attrs...)
	issues, err := s.inner.SearchIssues(ctx, query, filter)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(issues)))
	}
	s.done(ctx, span, t, err, attrs...)
	return issues, err
}

// ── Dependencies ────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) AddDependency(ctx context.Context, dep *types.Dependency, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.dep.from", dep.IssueID),
		attribute.String("bd.dep.to", dep.DependsOnID),
		attribute.String("bd.dep.type", string(dep.Type)),
	}
	ctx, span, t := s.op(ctx, "AddDependency", attrs...)
	err := s.inner.AddDependency(ctx, dep, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) RemoveDependency(ctx context.Context, issueID, dependsOnID string, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.dep.from", issueID),
		attribute.String("bd.dep.to", dependsOnID),
	}
	ctx, span, t := s.op(ctx, "RemoveDependency", attrs...)
	err := s.inner.RemoveDependency(ctx, issueID, dependsOnID, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetDependencies(ctx context.Context, issueID string) ([]*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetDependencies", attrs...)
	v, err := s.inner.GetDependencies(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetDependents(ctx context.Context, issueID string) ([]*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetDependents", attrs...)
	v, err := s.inner.GetDependents(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetDependenciesWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetDependenciesWithMetadata", attrs...)
	v, err := s.inner.GetDependenciesWithMetadata(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetDependentsWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetDependentsWithMetadata", attrs...)
	v, err := s.inner.GetDependentsWithMetadata(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetDependencyTree(ctx context.Context, issueID string, maxDepth int, showAllPaths bool, reverse bool) ([]*types.TreeNode, error) {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", issueID),
		attribute.Int("bd.max_depth", maxDepth),
	}
	ctx, span, t := s.op(ctx, "GetDependencyTree", attrs...)
	v, err := s.inner.GetDependencyTree(ctx, issueID, maxDepth, showAllPaths, reverse)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

// ── Labels ──────────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) AddLabel(ctx context.Context, issueID, label, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", issueID),
		attribute.String("bd.label", label),
	}
	ctx, span, t := s.op(ctx, "AddLabel", attrs...)
	err := s.inner.AddLabel(ctx, issueID, label, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) RemoveLabel(ctx context.Context, issueID, label, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", issueID),
		attribute.String("bd.label", label),
	}
	ctx, span, t := s.op(ctx, "RemoveLabel", attrs...)
	err := s.inner.RemoveLabel(ctx, issueID, label, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetLabels(ctx context.Context, issueID string) ([]string, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetLabels", attrs...)
	v, err := s.inner.GetLabels(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetIssuesByLabel(ctx context.Context, label string) ([]*types.Issue, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.label", label)}
	ctx, span, t := s.op(ctx, "GetIssuesByLabel", attrs...)
	v, err := s.inner.GetIssuesByLabel(ctx, label)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

// ── Work queries ─────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) GetReadyWork(ctx context.Context, filter types.WorkFilter) ([]*types.Issue, error) {
	ctx, span, t := s.op(ctx, "GetReadyWork")
	v, err := s.inner.GetReadyWork(ctx, filter)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(v)))
	}
	s.done(ctx, span, t, err)
	return v, err
}

func (s *InstrumentedStorage) GetBlockedIssues(ctx context.Context, filter types.WorkFilter) ([]*types.BlockedIssue, error) {
	ctx, span, t := s.op(ctx, "GetBlockedIssues")
	v, err := s.inner.GetBlockedIssues(ctx, filter)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(v)))
	}
	s.done(ctx, span, t, err)
	return v, err
}

func (s *InstrumentedStorage) GetEpicsEligibleForClosure(ctx context.Context) ([]*types.EpicStatus, error) {
	ctx, span, t := s.op(ctx, "GetEpicsEligibleForClosure")
	v, err := s.inner.GetEpicsEligibleForClosure(ctx)
	s.done(ctx, span, t, err)
	return v, err
}

// ── Comments & events ────────────────────────────────────────────────────────

func (s *InstrumentedStorage) AddIssueComment(ctx context.Context, issueID, author, text string) (*types.Comment, error) {
	attrs := []attribute.KeyValue{
		attribute.String("bd.issue.id", issueID),
		attribute.String("bd.actor", author),
	}
	ctx, span, t := s.op(ctx, "AddIssueComment", attrs...)
	v, err := s.inner.AddIssueComment(ctx, issueID, author, text)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetIssueComments(ctx context.Context, issueID string) ([]*types.Comment, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetIssueComments", attrs...)
	v, err := s.inner.GetIssueComments(ctx, issueID)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetEvents(ctx context.Context, issueID string, limit int) ([]*types.Event, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.issue.id", issueID)}
	ctx, span, t := s.op(ctx, "GetEvents", attrs...)
	v, err := s.inner.GetEvents(ctx, issueID, limit)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetAllEventsSince(ctx context.Context, since time.Time) ([]*types.Event, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.since", since.Format(time.RFC3339))}
	ctx, span, t := s.op(ctx, "GetAllEventsSince", attrs...)
	v, err := s.inner.GetAllEventsSince(ctx, since)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

// ── Statistics ───────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) GetStatistics(ctx context.Context) (*types.Statistics, error) {
	ctx, span, t := s.op(ctx, "GetStatistics")
	v, err := s.inner.GetStatistics(ctx)
	s.done(ctx, span, t, err)
	if err == nil && v != nil {
		// Record current issue counts as gauge snapshots, broken down by status.
		statusAttr := func(status string) metric.MeasurementOption {
			return metric.WithAttributes(attribute.String("status", status))
		}
		s.issueGauge.Record(ctx, int64(v.OpenIssues), statusAttr("open"))
		s.issueGauge.Record(ctx, int64(v.InProgressIssues), statusAttr("in_progress"))
		s.issueGauge.Record(ctx, int64(v.ClosedIssues), statusAttr("closed"))
		s.issueGauge.Record(ctx, int64(v.DeferredIssues), statusAttr("deferred"))
	}
	return v, err
}

// ── Configuration ────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) SetConfig(ctx context.Context, key, value string) error {
	attrs := []attribute.KeyValue{attribute.String("bd.config.key", key)}
	ctx, span, t := s.op(ctx, "SetConfig", attrs...)
	err := s.inner.SetConfig(ctx, key, value)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetConfig(ctx context.Context, key string) (string, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.config.key", key)}
	ctx, span, t := s.op(ctx, "GetConfig", attrs...)
	v, err := s.inner.GetConfig(ctx, key)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetAllConfig(ctx context.Context) (map[string]string, error) {
	ctx, span, t := s.op(ctx, "GetAllConfig")
	v, err := s.inner.GetAllConfig(ctx)
	s.done(ctx, span, t, err)
	return v, err
}

// ── Transactions ─────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) RunInTransaction(ctx context.Context, commitMsg string, fn func(tx storage.Transaction) error) error {
	ctx, span, t := s.op(ctx, "RunInTransaction", attribute.String("db.commit_msg", commitMsg))
	err := s.inner.RunInTransaction(ctx, commitMsg, fn)
	s.done(ctx, span, t, err)
	return err
}

// ── Entity operations (v8) ──────────────────────────────────────────────────

func (s *InstrumentedStorage) CreateEntity(ctx context.Context, entity *types.Entity) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.entity.id", entity.ID),
		attribute.String("bd.entity.type", entity.EntityType),
	}
	ctx, span, t := s.op(ctx, "CreateEntity", attrs...)
	err := s.inner.CreateEntity(ctx, entity)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetEntity(ctx context.Context, id string) (*types.Entity, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.entity.id", id)}
	ctx, span, t := s.op(ctx, "GetEntity", attrs...)
	v, err := s.inner.GetEntity(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) UpdateEntity(ctx context.Context, entity *types.Entity) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.entity.id", entity.ID),
		attribute.String("bd.entity.type", entity.EntityType),
	}
	ctx, span, t := s.op(ctx, "UpdateEntity", attrs...)
	err := s.inner.UpdateEntity(ctx, entity)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) DeleteEntity(ctx context.Context, id string) error {
	attrs := []attribute.KeyValue{attribute.String("bd.entity.id", id)}
	ctx, span, t := s.op(ctx, "DeleteEntity", attrs...)
	err := s.inner.DeleteEntity(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) SearchEntities(ctx context.Context, filters storage.EntityFilters) ([]*types.Entity, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.entity.type", filters.EntityType)}
	if filters.TextQuery != "" {
		attrs = append(attrs, attribute.String("bd.entity.text_query", filters.TextQuery))
	}
	ctx, span, t := s.op(ctx, "SearchEntities", attrs...)
	entities, err := s.inner.SearchEntities(ctx, filters)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(entities)))
	}
	s.done(ctx, span, t, err, attrs...)
	return entities, err
}

func (s *InstrumentedStorage) MergeEntities(ctx context.Context, sourceEntityID, targetEntityID, actor string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.entity.source", sourceEntityID),
		attribute.String("bd.entity.target", targetEntityID),
		attribute.String("bd.actor", actor),
	}
	ctx, span, t := s.op(ctx, "MergeEntities", attrs...)
	err := s.inner.MergeEntities(ctx, sourceEntityID, targetEntityID, actor)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) RetrieveMemory(ctx context.Context, query storage.MemoryQuery) (*storage.MemoryContext, error) {
	attrs := []attribute.KeyValue{
		attribute.String("bd.query.text", query.TextQuery),
		attribute.Int("bd.query.max_hops", query.MaxHops),
		attribute.Int("bd.query.top_k", query.TopK),
	}
	ctx, span, t := s.op(ctx, "RetrieveMemory", attrs...)
	result, err := s.inner.RetrieveMemory(ctx, query)
	s.done(ctx, span, t, err, attrs...)
	return result, err
}

// ── Relationship operations (v8) ────────────────────────────────────────────

func (s *InstrumentedStorage) CreateRelationship(ctx context.Context, rel *types.Relationship) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.rel.id", rel.ID),
		attribute.String("bd.rel.type", rel.RelationshipType),
		attribute.String("bd.rel.from", rel.SourceEntityID),
		attribute.String("bd.rel.to", rel.TargetEntityID),
	}
	ctx, span, t := s.op(ctx, "CreateRelationship", attrs...)
	err := s.inner.CreateRelationship(ctx, rel)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetRelationship(ctx context.Context, id string) (*types.Relationship, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.rel.id", id)}
	ctx, span, t := s.op(ctx, "GetRelationship", attrs...)
	v, err := s.inner.GetRelationship(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) UpdateRelationship(ctx context.Context, rel *types.Relationship) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.rel.id", rel.ID),
		attribute.String("bd.rel.type", rel.RelationshipType),
	}
	ctx, span, t := s.op(ctx, "UpdateRelationship", attrs...)
	err := s.inner.UpdateRelationship(ctx, rel)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) DeleteRelationship(ctx context.Context, id string) error {
	attrs := []attribute.KeyValue{attribute.String("bd.rel.id", id)}
	ctx, span, t := s.op(ctx, "DeleteRelationship", attrs...)
	err := s.inner.DeleteRelationship(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) SearchRelationships(ctx context.Context, filters storage.RelationshipFilters) ([]*types.Relationship, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.rel.type", filters.RelationshipType)}
	ctx, span, t := s.op(ctx, "SearchRelationships", attrs...)
	relationships, err := s.inner.SearchRelationships(ctx, filters)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(relationships)))
	}
	s.done(ctx, span, t, err, attrs...)
	return relationships, err
}

func (s *InstrumentedStorage) GetRelationshipsWithTemporalFilter(ctx context.Context, entityID string, validAt time.Time, direction storage.RelationshipDirection) ([]*types.Relationship, error) {
	attrs := []attribute.KeyValue{
		attribute.String("bd.entity.id", entityID),
		attribute.String("bd.valid_at", validAt.Format(time.RFC3339)),
	}
	ctx, span, t := s.op(ctx, "GetRelationshipsWithTemporalFilter", attrs...)
	relationships, err := s.inner.GetRelationshipsWithTemporalFilter(ctx, entityID, validAt, direction)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(relationships)))
	}
	s.done(ctx, span, t, err, attrs...)
	return relationships, err
}

// ── Episode operations (v8) ─────────────────────────────────────────────────

func (s *InstrumentedStorage) CreateEpisode(ctx context.Context, episode *types.Episode) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.episode.id", episode.ID),
		attribute.String("bd.episode.source", episode.Source),
	}
	ctx, span, t := s.op(ctx, "CreateEpisode", attrs...)
	err := s.inner.CreateEpisode(ctx, episode)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetEpisode(ctx context.Context, id string) (*types.Episode, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.episode.id", id)}
	ctx, span, t := s.op(ctx, "GetEpisode", attrs...)
	v, err := s.inner.GetEpisode(ctx, id)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) SearchEpisodes(ctx context.Context, filters storage.EpisodeFilters) ([]*types.Episode, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.episode.source", filters.Source)}
	ctx, span, t := s.op(ctx, "SearchEpisodes", attrs...)
	episodes, err := s.inner.SearchEpisodes(ctx, filters)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(episodes)))
	}
	s.done(ctx, span, t, err, attrs...)
	return episodes, err
}

// ── Ontology operations (v8) ────────────────────────────────────────────────

func (s *InstrumentedStorage) RegisterEntityType(ctx context.Context, schema *types.EntityTypeSchema) error {
	attrs := []attribute.KeyValue{attribute.String("bd.entity_type.name", schema.TypeName)}
	ctx, span, t := s.op(ctx, "RegisterEntityType", attrs...)
	err := s.inner.RegisterEntityType(ctx, schema)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) RegisterRelationshipType(ctx context.Context, schema *types.RelationshipTypeSchema) error {
	attrs := []attribute.KeyValue{attribute.String("bd.rel_type.name", schema.TypeName)}
	ctx, span, t := s.op(ctx, "RegisterRelationshipType", attrs...)
	err := s.inner.RegisterRelationshipType(ctx, schema)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) GetEntityTypes(ctx context.Context) ([]*types.EntityTypeSchema, error) {
	ctx, span, t := s.op(ctx, "GetEntityTypes")
	v, err := s.inner.GetEntityTypes(ctx)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(v)))
	}
	s.done(ctx, span, t, err)
	return v, err
}

func (s *InstrumentedStorage) GetRelationshipTypes(ctx context.Context) ([]*types.RelationshipTypeSchema, error) {
	ctx, span, t := s.op(ctx, "GetRelationshipTypes")
	v, err := s.inner.GetRelationshipTypes(ctx)
	if err == nil {
		span.SetAttributes(attribute.Int("bd.result.count", len(v)))
	}
	s.done(ctx, span, t, err)
	return v, err
}

func (s *InstrumentedStorage) GetEntityTypeSchema(ctx context.Context, typeName string) (*types.EntityTypeSchema, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.entity_type.name", typeName)}
	ctx, span, t := s.op(ctx, "GetEntityTypeSchema", attrs...)
	v, err := s.inner.GetEntityTypeSchema(ctx, typeName)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) GetRelationshipTypeSchema(ctx context.Context, typeName string) (*types.RelationshipTypeSchema, error) {
	attrs := []attribute.KeyValue{attribute.String("bd.rel_type.name", typeName)}
	ctx, span, t := s.op(ctx, "GetRelationshipTypeSchema", attrs...)
	v, err := s.inner.GetRelationshipTypeSchema(ctx, typeName)
	s.done(ctx, span, t, err, attrs...)
	return v, err
}

func (s *InstrumentedStorage) ValidateEntityAgainstType(ctx context.Context, entity *types.Entity, typeName string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.entity.id", entity.ID),
		attribute.String("bd.entity_type.name", typeName),
	}
	ctx, span, t := s.op(ctx, "ValidateEntityAgainstType", attrs...)
	err := s.inner.ValidateEntityAgainstType(ctx, entity, typeName)
	s.done(ctx, span, t, err, attrs...)
	return err
}

func (s *InstrumentedStorage) ValidateRelationshipAgainstType(ctx context.Context, rel *types.Relationship, typeName string) error {
	attrs := []attribute.KeyValue{
		attribute.String("bd.rel.id", rel.ID),
		attribute.String("bd.rel_type.name", typeName),
	}
	ctx, span, t := s.op(ctx, "ValidateRelationshipAgainstType", attrs...)
	err := s.inner.ValidateRelationshipAgainstType(ctx, rel, typeName)
	s.done(ctx, span, t, err, attrs...)
	return err
}

// ── Lifecycle ────────────────────────────────────────────────────────────────

func (s *InstrumentedStorage) Close() error {
	return s.inner.Close()
}
