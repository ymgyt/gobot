package app

import "github.com/juju/errors"

var (
	ErrUserNotFound = errors.New("user not found")
)

func IsUserNotFound(err error) bool {
	return errors.Cause(err) == ErrUserNotFound
}
