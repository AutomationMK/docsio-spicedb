package permissions

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed schema/schema.zed
var schema string

// initialize the production SpiceDB version af our Client interface
type spiceDBClient struct {
	raw *authzed.Client
}

// NewClient creates an authzed client connaction based on configuration
// three possible branches for testing, development, or defalt production
func NewClient(cfg ClientConfig) (Client, error) {
	if cfg.Endpoint == "" || cfg.Key == "" {
		return nil, errors.New("endpoint and key are required")
	}

	var opts []grpc.DialOption
	switch cfg.SecureType {
	case SecureTest:
		// test only - no tls, no verification
		opts = append(opts,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpcutil.WithInsecureBearerToken(cfg.Key),
		)
	case SecureDev:
		// dev only - tls handshake but no cert check.
		opts = append(opts,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})),
			grpcutil.WithBearerToken(cfg.Key),
		)
	default:
		// production tls 1.3, system root CAs.
		creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})
		opts = append(opts,
			grpc.WithTransportCredentials(creds),
			grpcutil.WithBearerToken(cfg.Key),
		)
	}

	raw, err := authzed.NewClient(cfg.Endpoint, opts...)
	if err != nil {
		return nil, err
	}
	return &spiceDBClient{raw: raw}, nil
}

// BootstrapSchema takes the embed schema/schema.zed file
// created at the top of this file and writes to the SpiceDB schema
func BootstrapSchema(ctx context.Context, c Client) error {
	s, ok := c.(*spiceDBClient)
	if !ok {
		return errors.New("permissions: cannot bootstrap unknown client")
	}
	_, err := s.raw.WriteSchema(ctx, &v1.WriteSchemaRequest{Schema: schema})
	return wrapSpiceDBError(err)
}

// Check takes a domain-typed CheckRequest and uses the spice proto api
// CheckPermission to either return a bool if permission is allowed and
// return a spicedb or grpc error if that happens
func (c *spiceDBClient) Check(ctx context.Context, req CheckRequest) (bool, error) {
	rt, rid, _ := strings.Cut(req.Resource, ":")
	st, sid, _ := strings.Cut(req.Subject, ":")

	consistency := &v1.Consistency{
		Requirement: &v1.Consistency_MinimizeLatency{MinimizeLatency: true},
	}
	if req.AtLeastAsFresh != "" {
		consistency = &v1.Consistency{
			Requirement: &v1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: &v1.ZedToken{Token: req.AtLeastAsFresh},
			},
		}
	}

	cpReq := &v1.CheckPermissionRequest{
		Resource: &v1.ObjectReference{
			ObjectType: rt,
			ObjectId:   rid,
		},
		Permission: req.Permission,
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: st,
				ObjectId:   sid,
			},
		},
		Consistency: consistency,
	}

	// check permission with 3 retries if the spicedb is unavailable or timeout
	var resp *v1.CheckPermissionResponse
	op := func() error {
		var err error
		resp, err = c.raw.CheckPermission(ctx, cpReq)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrUnavailable) || errors.Is(err, ErrTimeout) {
			return err // retryable
		}
		return backoff.Permanent(err)
	}
	// return spicedb error if past 3 retries
	if err := backoff.Retry(
		op,
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3),
	); err != nil {
		return false, wrapSpiceDBError(err)
	}

	// handle Permissionship from the spicedb response
	switch resp.Permissionship {
	case v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION:
		return true, nil
	case v1.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION:
		return false, nil
	case v1.CheckPermissionResponse_PERMISSIONSHIP_CONDITIONAL_PERMISSION:
		return false, ErrCaveatContextMissing
	default:
		return false, fmt.Errorf("unexpected permissionship: %v", resp.Permissionship)
	}
}

// Touch creates a new resource:relation@subject in the permissions database
// this wraps the write function but sets the specific operation code
func (c *spiceDBClient) Touch(ctx context.Context, resource,
	relation, subject string) (Token, error) {
	return c.write(ctx, v1.RelationshipUpdate_OPERATION_TOUCH,
		resource, relation, subject,
	)
}

// Delete deletes an resource:relation@subject in the permissions database
// this wraps the write function but sets the specific operation code
func (c *spiceDBClient) Delete(ctx context.Context, resource,
	relation, subject string) (Token, error) {
	return c.write(ctx, v1.RelationshipUpdate_OPERATION_DELETE,
		resource, relation, subject,
	)
}

// write handles any creat, update, or delete operations for
// resource:relation@subject in the permissions database depending
// on the operation code given
func (c *spiceDBClient) write(ctx context.Context, op v1.RelationshipUpdate_Operation,
	resource, relation, subject string) (Token, error) {
	rt, rid, _ := strings.Cut(resource, ":")
	st, sid, _ := strings.Cut(subject, ":")
	resp, err := c.raw.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: []*v1.RelationshipUpdate{{
			Operation: op,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: rt, ObjectId: rid},
				Relation: relation,
				Subject: &v1.SubjectReference{Object: &v1.ObjectReference{
					ObjectType: st, ObjectId: sid,
				}},
			},
		}},
	})
	if err != nil {
		return "", wrapSpiceDBError(err)
	}

	return Token(resp.WrittenAt.Token), nil
}

// LookupResource finds what resources can be seen based on permission
// of the subject given as a filter. This is a server-streaming rpc
// and therfore resources come back as streams in a for loop
func (c *spiceDBClient) LookupResource(ctx context.Context, subject,
	resourceType, permission string) ([]string, error) {
	st, sid, _ := strings.Cut(subject, ":")
	stream, err := c.raw.LookupResources(ctx, &v1.LookupResourcesRequest{
		ResourceObjectType: resourceType,
		Permission:         permission,
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{ObjectType: st, ObjectId: sid},
		},
	})
	if err != nil {
		return nil, wrapSpiceDBError(err)
	}

	var ids []string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, wrapSpiceDBError(err)
		}
		ids = append(ids, resp.ResourceObjectId)
	}

	return ids, nil
}
