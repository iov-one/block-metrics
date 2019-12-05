package store

import (
	"database/sql"

	"github.com/iov-one/weave/errors"
	"github.com/lib/pq"
)

var (
	// Explorer errors start from 2000

	// ErrConflict is returned when an operation cannot be completed
	// because of database constraints.
	ErrConflict = errors.Register(2000, "conflict")
	// ErrLimit is returned when allowed database query limit is exceeded
	ErrLimit = errors.Register(2001, "limit")
)

func wrapPgErr(err error, msg string) error {
	if err == nil {
		return nil
	}
	return errors.Wrap(castPgErr(err), msg)
}

func castPgErr(err error) error {
	if err == nil {
		return nil
	}

	if err == sql.ErrNoRows {
		return errors.ErrNotFound
	}

	if e, ok := err.(*pq.Error); ok {
		switch prefix := e.Code[:2]; prefix {
		case "20":
			return errors.Wrap(errors.ErrNotFound, e.Message)
		case "23":
			return errors.Wrap(ErrConflict, e.Message)
		}
		err = errors.Wrap(err, string(e.Code))
	}

	return err
}
