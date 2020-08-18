package metrics

import (
	"github.com/iov-one/weave/errors"
)

var (
	// Explorer errors start from 2100

	ErrNotImplemented = errors.Register(2100, "not implemented")
	ErrFailedResponse = errors.Register(2002, "failed response")
)
