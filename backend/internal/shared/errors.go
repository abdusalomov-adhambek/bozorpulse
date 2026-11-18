// Package shared holds cross-cutting kernel types used by every module.
package shared

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
	ErrInvalid      = errors.New("invalid input")
	ErrNotApproved  = errors.New("account is not approved yet")
)

// DomainError carries an HTTP status alongside a domain error.
type DomainError struct {
	Err    error
	Status int
}

func (e DomainError) Error() string { return e.Err.Error() }
func (e DomainError) Unwrap() error { return e.Err }

func WithStatus(err error, status int) error {
	return DomainError{Err: err, Status: status}
}
