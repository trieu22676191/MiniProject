package grpcadapter

import (
	"fmt"
	"testing"

	"github.com/trieu22676191/MiniProject/be/services/order/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/order/internal/usecase"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToStatus_AC02_MapsForbiddenToPermissionDenied(t *testing.T) {
	err := toStatus(usecase.ErrForbidden)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("code = %v, want %v", st.Code(), codes.PermissionDenied)
	}
	assertReason(t, st, "PERMISSION_DENIED")
}

func TestToStatus_AC03_MapsSelfApprovalToPermissionDeniedWithReason(t *testing.T) {
	err := toStatus(domain.ErrSelfApproval)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("code = %v, want %v", st.Code(), codes.PermissionDenied)
	}
	assertReason(t, st, "SELF_APPROVAL")
}

func TestToStatus_AC09_MapsInvalidTransitionToFailedPreconditionWithReason(t *testing.T) {
	err := toStatus(domain.ErrInvalidTransition)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %v, want %v", st.Code(), codes.FailedPrecondition)
	}
	assertReason(t, st, "INVALID_TRANSITION")
}

func TestToStatus_AC01_UnwrapsUpstreamNotFoundFromStock(t *testing.T) {
	upstreamErr := status.Error(codes.NotFound, "one or more items not found")
	wrappedErr := fmt.Errorf("get prices from stock: %w", upstreamErr)

	err := toStatus(wrappedErr)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("code = %v, want %v", st.Code(), codes.NotFound)
	}
}

func assertReason(t *testing.T, st *status.Status, wantReason string) {
	t.Helper()
	for _, detail := range st.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			if info.Reason != wantReason {
				t.Fatalf("reason = %s, want %s", info.Reason, wantReason)
			}
			return
		}
	}
	t.Fatalf("expected ErrorInfo detail with reason %s, but none found in %v", wantReason, st.Details())
}
