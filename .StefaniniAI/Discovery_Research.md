# Discovery Research - Beads Codebase

*Generated: 2026-03-16*

## Root Files Analyzed
- go.mod: Module `github.com/steveyegge/beads`, Go 1.25.8, Cobra CLI, Dolt driver, OTel, Charm TUI, Anthropic SDK
- Makefile: Build targets for `bd` binary, CGO required, test/bench/install/fmt targets
- flake.nix: Nix packaging for 4 platforms (aarch64-darwin/linux, x86_64-darwin/linux)
- codecov.yml: Components: cli-commands (cmd/**), storage-engine (internal/storage/**)

## Entry Points
- cmd/bd/main.go: Root Cobra command (`bd`), 100+ subcommands, global state (dbPath, store, jsonOutput, hookRunner)
- beads.go: Public Go API re-exporting internal types (Storage, Issue, Status, etc.)
- integrations/beads-mcp/: Python FastMCP server for AI assistant integration

## Internal Packages (30+)
- beads/: Workspace discovery, git integration, redirect handling
- types/: Core domain models (Issue with 70+ DB columns, Dependency, Label, Comment, Event)
- storage/: Interface definition (Storage, Transaction, DoltStorage)
- storage/dolt/: Dolt MySQL implementation, schema v7, 11 migrations
- config/: Viper-based YAML config with 4-level precedence
- configfile/: metadata.json parsing for Dolt mode config
- hooks/: Event hooks (on_create, on_update, on_close)
- audit/: Append-only JSONL interaction log
- telemetry/: OpenTelemetry integration (opt-in)
- ui/: Terminal styling, markdown rendering, paging
- query/: Expression parser with SQL compilation + in-memory predicates
- routing/: Multi-repo context switching with prefix-to-path routes
- recipes/: Formula/template system (12 built-in integrations)
- molecules/: Compound issues (bonded from constituents)
- compact/: AI-powered semantic memory decay via Claude Haiku
- lockfile/: Process-level mutual exclusion
- git/: Git/Jujutsu VCS integration
- github/, gitlab/, jira/, linear/: External tracker integrations
- doltserver/: Dolt server lifecycle (ephemeral ports, shared mode)
- idgen/: SHA256-based hash IDs with configurable prefix/length
- tracker/: Bidirectional sync plugin architecture
- timeparsing/: Natural language date parsing
- validation/: ID prefix, priority, type validation
- debug/: Debug logging utilities
- testutil/: Dolt container setup for integration tests

## Schema Tables (14)
issues, dependencies, labels, comments, events, config, metadata,
child_counters, issue_snapshots, compaction_snapshots, repo_mtimes,
routes, issue_counter, interactions, federation_peers, wisps

## Key Architectural Patterns
1. Two-layer model: CLI → Dolt Database → Remote (push/pull)
2. Dual connection modes: Embedded (in-process) vs Server (TCP)
3. Hash-based IDs for distributed collision prevention
4. Interface-based storage with transaction support
5. Hook-based extensibility for tracker adapters
6. Two-tier query evaluation (SQL for simple, in-memory for complex)
7. Append-only audit trail + OTel instrumentation
8. Semantic compaction via AI summarization
9. Wisp/Molecule system for ephemeral local-only workflows
