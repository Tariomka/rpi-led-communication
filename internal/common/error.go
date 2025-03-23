package common

import "errors"

var (
	ErrOutOfBounds          = errors.New("index out of bounds")
	ErrDhcpRequestFailed    = errors.New("DHCP did not complete")
	ErrNoInternetConnection = errors.New("did not connect to internet")
	ErrServerNotInitialized = errors.New("server not initialized")
	ErrNoListener           = errors.New("no listener provided")
)
