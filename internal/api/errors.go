package api

import "errors"

var (
	errInvalidDate   = errors.New("invalid date: expected format YYYY-MM-DD, for example 2024-02-29")
	errDuplicateDate = errors.New("date parameter must be given at most once")
)
