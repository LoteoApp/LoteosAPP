package users

import "strings"

// trimIfPresent trims s when it's present (non-nil), leaving an absent
// field (nil, meaning "unchanged") as nil.
func trimIfPresent(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// isBlank reports whether s is present but empty — i.e. the caller sent the
// field but it has no content, as opposed to not sending it at all.
func isBlank(s *string) bool {
	return s != nil && *s == ""
}
