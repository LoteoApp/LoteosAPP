package domain

// Kind classifies a business error by the kind of failure it represents,
// independent of any delivery mechanism (HTTP, gRPC, CLI, ...).
type Kind int

const (
	KindInvalid Kind = iota
	KindForbidden
	KindConflict
	KindNotFound
)

// Error is a business error with a stable Code and a user-facing Message,
// classified by Kind so adapters can map it to their own representation
// (e.g. an HTTP status) without domain knowing about that representation.
type Error struct {
	Kind    Kind
	Code    string
	Message string
}

func (err *Error) Error() string {
	return err.Message
}
