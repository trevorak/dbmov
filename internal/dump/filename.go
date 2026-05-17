package dump

import "strings"

// DatabaseFromDumpFilename returns the database name from a dump basename (e.g. foo.sql, foo.sql.gz).
// For unrecognized suffixes it returns an empty string.
func DatabaseFromDumpFilename(base string) string {
	switch {
	case strings.HasSuffix(base, ".sql.gz"):
		return strings.TrimSuffix(base, ".sql.gz")
	case strings.HasSuffix(base, ".sql"):
		return strings.TrimSuffix(base, ".sql")
	default:
		return ""
	}
}
