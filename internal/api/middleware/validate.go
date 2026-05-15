package middleware

import (
	"context"

	protovalidate "buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ProtoValidate returns a gRPC unary interceptor that enforces buf.validate
// rules declared in proto field options. Validation is applied before the
// handler so service code never receives an invalid request.
func ProtoValidate() grpc.UnaryServerInterceptor {
	v, err := protovalidate.New()
	if err != nil {
		panic("protovalidate: failed to create validator: " + err.Error())
	}
	return func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			if err := v.Validate(msg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
			}
		}
		return handler(ctx, req)
	}
}

// ProtoValidateStream returns a gRPC stream interceptor that validates the
// first (client-sent) message of a streaming RPC.
// For server-streaming RPCs the request struct is validated once at stream open.
func ProtoValidateStream() grpc.StreamServerInterceptor {
	v, err := protovalidate.New()
	if err != nil {
		panic("protovalidate: failed to create validator: " + err.Error())
	}
	return func(
		srv any,
		ss grpc.ServerStream,
		_ *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Wrap the stream to intercept the first RecvMsg and validate it.
		return handler(srv, &validatingStream{ServerStream: ss, validator: v})
	}
}

type validatingStream struct {
	grpc.ServerStream
	validator protovalidate.Validator
	validated bool
}

func (s *validatingStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if !s.validated {
		s.validated = true
		if msg, ok := m.(proto.Message); ok {
			if err := s.validator.Validate(msg); err != nil {
				return status.Errorf(codes.InvalidArgument, "validation: %v", err)
			}
		}
	}
	return nil
}
