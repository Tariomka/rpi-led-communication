package common

import "errors"

var (
	ErrDhcpRequestFailed    = errors.New("DHCP did not complete")
	ErrNoInternetConnection = errors.New("did not connect to internet")
	ErrNoListener           = errors.New("no listener provided")
	ErrRebootCommand        = errors.New("ReBoot is only for the vendor upload tool")
	ErrNoUpload             = errors.New("no show upload running")
	ErrUploadOffset         = errors.New("show data chunk at an unexpected offset")
)
