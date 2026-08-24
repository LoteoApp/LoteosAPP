package domain

// Kind classifies a business error by the kind of failure it represents,
// independent of any delivery mechanism (HTTP, gRPC, CLI, ...).
type Kind int

const (
	KindInvalid Kind = iota
	KindForbidden
	KindConflict
	KindNotFound
	KindUnavailable
)

// Error is a business error with a stable Code and a user-facing Message,
// classified by Kind so adapters can map it to their own representation
// (e.g. an HTTP status) without domain knowing about that representation.
// Cause, when set, is the underlying error that triggered it (e.g. a
// PostgreSQL connectivity failure); adapters may log it, but Message is
// what's safe to show to the caller.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Cause   error
}

func (err *Error) Error() string {
	return err.Message
}

func (err *Error) Unwrap() error {
	return err.Cause
}
