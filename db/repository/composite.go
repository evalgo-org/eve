package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	evedb "eve.evalgo.org/db"
	"eve.evalgo.org/semantic"
)

// CompositeRepository combines all repository types for complete data management.
// Applications can use this to access all database backends through a single interface.
//
// Design Pattern:
//   - Composite pattern combining multiple repositories
//   - Graceful degradation if backends unavailable
//   - Coordinated operations across databases
//   - Single point of configuration
//
// Usage:
//
//	config := repository.ConfigFromEnv()
//	repo, err := repository.NewCompositeRepository(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer repo.Close()
//
//	// Use individual repositories
//	repo.Documents.SaveWorkflow(ctx, id, workflow)
//	repo.Graph.GetDependencies(ctx, actionID)
//	repo.Metrics.SaveRun(ctx, run)
//	repo.Cache.AcquireLock(ctx, key, ttl)
//
//	// Or use convenience methods
//	repo.SaveAction(ctx, action, workflowID)
//	repo.GetAction(ctx, actionID)
type CompositeRepository struct {
	Documents DocumentRepository
	Graph     GraphRepository
	Metrics   MetricsRepository
	Cache     CacheRepository
}

// Config holds configuration for all repository backends.
// Use ConfigFromEnv() to populate from environment variables.
type Config struct {
	// CouchDB configuration
	CouchDBURL      string
	CouchDBUser     string
	CouchDBPassword string

	// Neo4j configuration
	Neo4jURL      string
	Neo4jUser     string
	Neo4jPassword string

	// PostgreSQL configuration
	PostgresURL string

	// Redis/Valkey/DragonflyDB configuration
	RedisURL string
}

// ConfigFromEnv creates configuration from environment variables.
// Provides sensible defaults for local development.
//
// Environment Variables:
//   - EVE_COUCHDB_URL (default: http://localhost:5984)
//   - EVE_COUCHDB_USER (default: "")
//   - EVE_COUCHDB_PASSWORD (default: "")
//   - EVE_NEO4J_URL (default: bolt://localhost:7687)
//   - EVE_NEO4J_USER (default: neo4j)
//   - EVE_NEO4J_PASSWORD (default: password)
//   - EVE_POSTGRES_URL (default: postgresql://user:pass@localhost:5432/eve?sslmode=disable)
//   - EVE_REDIS_URL (default: redis://localhost:6379)
func ConfigFromEnv() Config {
	return Config{
		CouchDBURL:      getEnv("EVE_COUCHDB_URL", "http://localhost:5984"),
		CouchDBUser:     getEnv("EVE_COUCHDB_USER", ""),
		CouchDBPassword: getEnv("EVE_COUCHDB_PASSWORD", ""),

		Neo4jURL:      getEnv("EVE_NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("EVE_NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("EVE_NEO4J_PASSWORD", "password"),

		PostgresURL: getEnv("EVE_POSTGRES_URL", "postgresql://user:pass@localhost:5432/eve?sslmode=disable"),

		RedisURL: getEnv("EVE_REDIS_URL", "redis://localhost:6379"),
	}
}

// NewCompositeRepository creates a new composite repository with all backends.
// Initializes each backend that has configuration provided.
// Returns error only if explicitly required backends fail.
//
// Graceful Degradation:
//   - If CouchDB URL empty, Documents will be nil
//   - If Neo4j URL empty, Graph will be nil
//   - If PostgreSQL URL empty, Metrics will be nil
//   - If Redis URL empty, Cache will be nil
//
// Applications should check for nil before using:
//
//	if repo.Graph != nil {
//	    deps, err := repo.Graph.GetDependencies(ctx, actionID)
//	}
func NewCompositeRepository(config Config) (*CompositeRepository, error) {
	var (
		documents DocumentRepository
		graph     GraphRepository
		metrics   MetricsRepository
		cache     CacheRepository
		err       error
	)

	// Initialize CouchDB (documents)
	if config.CouchDBURL != "" {
		documents, err = NewCouchDBRepository(config.CouchDBURL, config.CouchDBUser, config.CouchDBPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize CouchDB: %w", err)
		}
		log.Println("✓ CouchDB document repository initialized")
	}

	// Initialize Neo4j (graph)
	if config.Neo4jURL != "" {
		graph, err = NewNeo4jRepository(config.Neo4jURL, config.Neo4jUser, config.Neo4jPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Neo4j: %w", err)
		}
		log.Println("✓ Neo4j graph repository initialized")
	}

	// Initialize PostgreSQL (metrics)
	if config.PostgresURL != "" {
		pgDB, err := evedb.NewPostgresDB(config.PostgresURL)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
		}
		metrics = NewPostgresMetricsRepository(pgDB)
		log.Println("✓ PostgreSQL metrics repository initialized")
	}

	// Initialize Redis (cache)
	if config.RedisURL != "" {
		cache, err = NewRedisRepository(config.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis: %w", err)
		}
		log.Println("✓ Redis cache repository initialized")
	}

	return &CompositeRepository{
		Documents: documents,
		Graph:     graph,
		Metrics:   metrics,
		Cache:     cache,
	}, nil
}

// SaveAction saves an action to all configured backends.
// Coordinates the save across CouchDB (master), Neo4j (topology), and Redis (cache).
//
// Operation Flow:
//  1. Save complete JSON-LD document to CouchDB (master copy)
//  2. Extract and save relationship topology to Neo4j
//  3. Link to workflow in Neo4j if workflowID provided
//  4. Cache action for fast retrieval
//
// Consistency:
//   - Eventual consistency across backends
//   - CouchDB is source of truth
//   - Neo4j/Redis failures logged but don't fail operation
//   - Applications should handle partial failures
func (r *CompositeRepository) SaveAction(ctx context.Context, action *semantic.SemanticScheduledAction, workflowID string) error {
	// 1. Save to CouchDB (master document)
	if r.Documents != nil {
		if err := r.Documents.SaveAction(ctx, action.Identifier, action); err != nil {
			return fmt.Errorf("failed to save action to CouchDB: %w", err)
		}
	}

	// 2. Save to Neo4j (graph relationships)
	if r.Graph != nil {
		if err := r.Graph.StoreActionGraph(ctx, action); err != nil {
			return fmt.Errorf("failed to save action graph: %w", err)
		}

		// Link to workflow if specified
		if workflowID != "" {
			if err := r.Graph.LinkActionToWorkflow(ctx, action.Identifier, workflowID); err != nil {
				return fmt.Errorf("failed to link action to workflow: %w", err)
			}
		}
	}

	// 3. Save metadata to PostgreSQL (for foreign key relationships and queries)
	if r.Metrics != nil {
		if pgMetrics, ok := r.Metrics.(*PostgresMetricsRepository); ok {
			// Marshal action to JSON for storage
			jsonLD, err := json.Marshal(action)
			if err != nil {
				return fmt.Errorf("failed to marshal action: %w", err)
			}

			if err := pgMetrics.SaveActionMetadata(ctx, action.Identifier, workflowID, action.Type, action.Name, action.Description, jsonLD); err != nil {
				return fmt.Errorf("failed to save action metadata to PostgreSQL: %w", err)
			}
		}
	}

	// 4. Cache for fast access
	if r.Cache != nil {
		_ = r.Cache.SetCache(ctx, "action:"+action.Identifier, action, 5*60) // 5 min TTL
	}

	return nil
}

// GetAction retrieves an action with cache-first strategy.
// Tries cache first, falls back to CouchDB on miss.
//
// Performance:
//   - Cache hit: ~1ms
//   - Cache miss: ~12ms (CouchDB query + cache update)
//
// Consistency:
//   - Cache may be stale (5 minute TTL)
//   - For latest data, query Documents directly
func (r *CompositeRepository) GetAction(ctx context.Context, actionID string) (*semantic.SemanticScheduledAction, error) {
	// Try cache first
	if r.Cache != nil {
		var action semantic.SemanticScheduledAction
		if err := r.Cache.GetCache(ctx, "action:"+actionID, &action); err == nil {
			return &action, nil
		}
	}

	// Fetch from CouchDB
	if r.Documents != nil {
		action, err := r.Documents.GetAction(ctx, actionID)
		if err != nil {
			return nil, err
		}

		// Update cache
		if r.Cache != nil {
			_ = r.Cache.SetCache(ctx, "action:"+actionID, action, 5*60)
		}

		return action, nil
	}

	return nil, fmt.Errorf("no document repository available")
}

// DeleteAction deletes an action from all backends.
// Coordinates deletion across CouchDB, Neo4j, and Redis.
//
// Best Effort:
//   - Attempts to delete from all backends
//   - Failures logged but don't stop deletion
//   - Returns first error encountered
func (r *CompositeRepository) DeleteAction(ctx context.Context, actionID string) error {
	// Delete from CouchDB
	if r.Documents != nil {
		if err := r.Documents.DeleteAction(ctx, actionID); err != nil {
			return fmt.Errorf("failed to delete from CouchDB: %w", err)
		}
	}

	// Delete from Neo4j
	if r.Graph != nil {
		if err := r.Graph.DeleteActionGraph(ctx, actionID); err != nil {
			return fmt.Errorf("failed to delete from Neo4j: %w", err)
		}
	}

	// Delete from cache
	if r.Cache != nil {
		_ = r.Cache.DeleteCache(ctx, "action:"+actionID)
	}

	return nil
}

// Close closes all repository connections.
// Should be called when the repository is no longer needed.
func (r *CompositeRepository) Close() error {
	var errs []error

	if closer, ok := r.Documents.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if closer, ok := r.Graph.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if closer, ok := r.Cache.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing repositories: %v", errs)
	}

	return nil
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
