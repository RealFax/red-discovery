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

var (
	ErrConnPoolClosed    = errors.New("sdr: failed alloc grpc conn, cause conn pool closed")
	ErrCloseReadonlyConn = errors.New("sdr: failed close grpc conn, cause readonly")
)
