package cli

import (
	"context"

	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"google.golang.org/grpc/metadata"
)

// noopStream satisfies grpc.ServerStreamingServer[*runv1.RunEvent] without
// a real gRPC connection. All RunEvent messages are discarded; it is used
// when the reports.Reporter interface handles all output instead.
type noopStream struct {
	ctx context.Context
}

func (s *noopStream) Send(_ *runv1.RunEvent) error { return s.ctx.Err() }
func (s *noopStream) Context() context.Context     { return s.ctx }
func (s *noopStream) SetHeader(metadata.MD) error  { return nil }
func (s *noopStream) SendHeader(metadata.MD) error { return nil }
func (s *noopStream) SetTrailer(metadata.MD)       {}
func (s *noopStream) SendMsg(_ any) error          { return nil }
func (s *noopStream) RecvMsg(_ any) error          { return nil }
