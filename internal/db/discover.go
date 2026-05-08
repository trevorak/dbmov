package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
)

// DefaultExcluded contains schema names skipped unless overridden (e.g. --include-mysql).
var DefaultExcluded = map[string]struct{}{
	"information_schema": {},
	"performance_schema": {},
	"sys":                {},
	"mysql":              {},
}

// FilterDiscoveredNames applies default exclusions and glob filters to a raw SHOW DATABASES list.
func FilterDiscoveredNames(all []string, includeMySQL bool, includeGlobs, excludeGlobs []string) ([]string, error) {
	excl := make(map[string]struct{})
	for k, v := range DefaultExcluded {
		excl[k] = v
	}
	if includeMySQL {
		delete(excl, "mysql")
	}

	var filtered []string
outer:
	for _, name := range all {
		if _, skip := excl[name]; skip {
			continue
		}
		for _, g := range excludeGlobs {
			ok, err := filepath.Match(g, name)
			if err != nil {
				return nil, fmt.Errorf("exclude glob %q: %w", g, err)
			}
			if ok {
				continue outer
			}
		}
		filtered = append(filtered, name)
	}

	if len(includeGlobs) > 0 {
		var narrowed []string
		for _, name := range filtered {
			for _, g := range includeGlobs {
				ok, err := filepath.Match(g, name)
				if err != nil {
					return nil, fmt.Errorf("include glob %q: %w", g, err)
				}
				if ok {
					narrowed = append(narrowed, name)
					break
				}
			}
		}
		filtered = narrowed
	}

	sort.Strings(filtered)
	return filtered, nil
}

// DiscoverDatabases returns database names visible to the current user after applying filters.
func DiscoverDatabases(ctx context.Context, db *sql.DB, includeMySQL bool, includeGlobs, excludeGlobs []string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		all = append(all, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return FilterDiscoveredNames(all, includeMySQL, includeGlobs, excludeGlobs)
}
