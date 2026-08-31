package handler

import "regexp"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUUID reports whether id looks like a canonical UUID — the format
// PostgreSQL's uuid type accepts for a `::uuid` cast. A handler that forwards
// a path param straight into such a cast must check this first: otherwise a
// non-UUID value reaches PostgreSQL as a malformed cast and surfaces as an
// unclassified 500 instead of a 400.
func isValidUUID(id string) bool {
	return uuidPattern.MatchString(id)
}
