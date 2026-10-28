package common

import (
	"errors"

	"github.com/Tariomka/led-common-lib/pkg/common"
)

var (
	ErrDhcpRequestFailed    = errors.New("DHCP did not complete")
	ErrNoInternetConnection = errors.New("did not connect to internet")
	ErrNoListener           = errors.New("no listener provided")
	ErrNoWifiConfig         = errors.New("no Wi-Fi SSID in config.json")
	ErrRebootCommand        = errors.New("ReBoot is only for the vendor upload tool")
	ErrNoUpload             = errors.New("no show upload running")
	ErrUploadOffset         = errors.New("show data chunk at an unexpected offset")
)

// Aliases

var (
	ErrUnsupportedVersion = common.ErrUnsupportedVersion
	ErrPayloadTooLarge    = common.ErrPayloadTooLarge
	ErrUploadBusy         = common.ErrUploadBusy
	ErrUploadSize         = common.ErrUploadSize
)
