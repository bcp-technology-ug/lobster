package middleware_test

import (
	"context"
	"testing"

	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/api/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// passthroughHandler is a gRPC unary handler that always returns nil.
func passthroughHandler(_ context.Context, req any) (any, error) {
	return req, nil
}

func TestProtoValidate_validRequest(t *testing.T) {
	t.Parallel()
	interceptor := middleware.ProtoValidate()
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId: "ws-test",
		},
	}
	_, err := interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, passthroughHandler)
	if err != nil {
		t.Errorf("expected no error for valid request, got: %v", err)
	}
}

func TestProtoValidate_invalidRequest_missingSelectorRequired(t *testing.T) {
	t.Parallel()
	interceptor := middleware.ProtoValidate()
	// RunSyncRequest.selector has required = true; nil selector should fail.
	req := &runv1.RunSyncRequest{}
	_, err := interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, passthroughHandler)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code: got %v want %v", st.Code(), codes.InvalidArgument)
	}
}

func TestProtoValidate_nonProtoMessage(t *testing.T) {
	t.Parallel()
	interceptor := middleware.ProtoValidate()
	// Non-proto messages should pass through without validation.
	type plain struct{ Value string }
	req := &plain{Value: "hello"}
	resp, err := interceptor(context.Background(), req, &grpc.UnaryServerInfo{}, passthroughHandler)
	if err != nil {
		t.Errorf("expected no error for non-proto message, got: %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response for passthrough")
	}
}
