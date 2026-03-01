package error

import "errors"

var (
	ErrUserNotAuthenticated = errors.New("user not authenticated")
)
