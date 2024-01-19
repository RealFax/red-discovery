package discovery

import "github.com/pkg/errors"

var (
	ErrServiceNotExist      = errors.New("sdr: service not existed")
	ErrDiscoveryHasExist    = errors.New("sdr: discovery has existed")
	ErrShouldDiscoveryFirst = errors.New("sdr: should discovery first")
	ErrServiceUnreachable   = errors.New("sdr: service unreachable")
)

var (
	ErrInvalidEndpointPathFormat = errors.New("sdr: ParseEndpointPath invalid endpoint path format")
)
