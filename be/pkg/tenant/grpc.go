package tenant

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity")
	}
	tenantID, err := uuid.Parse(first(md.Get("x-tenant-id")))
	if err != nil || first(md.Get("x-user-id")) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing identity")
	}
	caps := make(map[string]bool)
	for _, cap := range strings.Split(first(md.Get("x-capabilities")), ",") {
		if cap = strings.TrimSpace(cap); cap != "" {
			caps[cap] = true
		}
	}
	identity := Identity{
		TenantID: tenantID, UserID: first(md.Get("x-user-id")),
		ServiceID: first(md.Get("x-service-id")), Capabilities: caps,
	}
	return handler(WithIdentity(ctx, identity), req)
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
