package socket5

import (
	"errors"
)

var (
	ErrReadHeader  = errors.New("read header[ver, nmethods] error")
	ErrAuthVersion = errors.New("invalid version")
	ErrReadMethods = errors.New("read methods error")
)
