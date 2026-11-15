package permissions

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client defines the Authorization client which can be mocked for tests
// or used for real production as long as in implements the definition
type Client interface {
	Check(ctx context.Context, req CheckRequest) (bool, error)
	Touch(ctx context.Context, resource, relation, subject string) (Token, error)
	Delete(ctx context.Context, resource, relation, subject string) (Token, error)
	LookupResource(ctx context.Context, subject, resourseType, permission string) ([]string, error)
}

// CheckRequest is the model used for sending check request info
type CheckRequest struct {
	Subject        string
	Resource       string
	Permission     string
	AtLeastAsFresh string
}

type Token string

// SecureFlag is used for case switching the gRPC request type
type SecureFlag int

// Global Enumeration values for SecureFlag types
const (
	SecureProd SecureFlag = iota // 0
	SecureDev                    // 1
	SecureTest                   // 2
)

// ClientConfig holds SpiceDB client connection information
type ClientConfig struct {
	Endpoint   string
	Key        string
	SecureType SecureFlag
}

// sentinel errors the rest of the app can compare against
var (
	ErrBadRequest           = errors.New("permissions: bad request")
	ErrNotFound             = errors.New("permissions: not found")
	ErrConflict             = errors.New("permissions: conflict")
	ErrCaveatContextMissing = errors.New("permissions: caveat context missing")
	ErrUnauthenticated      = errors.New("permissions: unauthenticated")
	ErrTimeout              = errors.New("permissions: timeout")
	ErrUnavailable          = errors.New("permissions: unavailable")
	ErrInternal             = errors.New("permissions: internal")
)

// wrapSpiceDBError translates a gRPC error into one of the
// sentinel errors above preserving the original message for debugging
func wrapSpiceDBError(err error) error {
	if err == nil {
		return nil
	}

	s, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}

	switch s.Code() {
	case codes.OK:
		return nil
	case codes.InvalidArgument, codes.NotFound:
		return fmt.Errorf("%w: %s", ErrBadRequest, s.Message())
	case codes.FailedPrecondition:
		return fmt.Errorf("%w: %s", ErrConflict, s.Message())
	case codes.Unauthenticated:
		return fmt.Errorf("%w: %s", ErrUnauthenticated, s.Message())
	case codes.DeadlineExceeded:
		return fmt.Errorf("%w: %s", ErrTimeout, s.Message())
	case codes.Unavailable:
		return fmt.Errorf("%w: %s", ErrUnavailable, s.Message())
	default:
		return fmt.Errorf("%w: code=%v %s", ErrInternal, s.Code(), s.Message())
	}
}
