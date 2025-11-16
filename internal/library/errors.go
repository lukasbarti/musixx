package library

import "errors"

// ErrNotFound indicates that a requested resource was not found in the repository.
var ErrNotFound = errors.New("library: not found")
