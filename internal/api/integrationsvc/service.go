// Package integrationsvc implements the gRPC IntegrationService server.
package integrationsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bcp-technology/lobster/internal/api/convert"
	"github.com/bcp-technology/lobster/internal/store"

	integrationsv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/integrations"
	integrationstore "github.com/bcp-technology/lobster/gen/sqlc/integrations"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdapterValidator is the interface the IntegrationService uses to validate
// a live adapter connection. Batch 2 (integrations package) will implement.
type AdapterValidator interface {
	Validate(ctx context.Context, adapterID string) (bool, error)
}

// StateNotifier is an optional interface that IntegrationService calls after a
// state change so in-process consumers (e.g. the runner's adapter registry)
// stay in sync with the DB.
type StateNotifier interface {
	Enable(id string)
	Disable(id string)
}

// Service implements integrationsv1.IntegrationServiceServer.
type Service struct {
	integrationsv1.UnimplementedIntegrationServiceServer

	store     *store.Store
	validator AdapterValidator
	notifier  StateNotifier
}

// New creates a Service. validator may be nil; ValidateIntegrationAdapter will
// return Unimplemented until it is wired.
func New(st *store.Store, validator AdapterValidator) *Service {
	return &Service{store: st, validator: validator}
}

// WithNotifier attaches a StateNotifier that is called whenever an adapter's
// enabled state changes. Returns the receiver for chaining.
func (s *Service) WithNotifier(n StateNotifier) *Service {
	s.notifier = n
	return s
}

// ListIntegrationAdapters returns a paginated list of adapters.
func (s *Service) ListIntegrationAdapters(ctx context.Context, req *integrationsv1.ListIntegrationAdaptersRequest) (*integrationsv1.ListIntegrationAdaptersResponse, error) {
	pageSize := convert.PageSizeOrDefault(req.PageSize)

	cursorUpdatedAt, cursorAdapterID, err := convert.DecodeCursor(req.PageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	rows, err := s.store.Integrations.ListIntegrationAdaptersPage(ctx, integrationstore.ListIntegrationAdaptersPageParams{
		CursorUpdatedAt: convert.PtrStrToInterface(cursorUpdatedAt),
		CursorAdapterID: cursorAdapterID,
		PageSize:        pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list adapters: %v", err)
	}

	adapters := make([]*integrationsv1.IntegrationAdapter, 0, len(rows))
	for _, r := range rows {
		adapter := convert.IntegrationAdapterFromDB(r)
		caps, capErr := s.store.Integrations.ListIntegrationAdapterCapabilities(ctx, r.AdapterID)
		if capErr == nil {
			for _, c := range caps {
				adapter.Capabilities = append(adapter.Capabilities, convert.AdapterCapabilityFromDB(c))
			}
		}
		adapters = append(adapters, adapter)
	}

	var nextToken string
	if int64(len(rows)) == pageSize {
		last := rows[len(rows)-1]
		nextToken = convert.EncodeCursor(last.UpdatedAt, last.AdapterID)
	}

	return &integrationsv1.ListIntegrationAdaptersResponse{
		Adapters:      adapters,
		NextPageToken: nextToken,
	}, nil
}

// GetIntegrationAdapter returns a single adapter by ID.
func (s *Service) GetIntegrationAdapter(ctx context.Context, req *integrationsv1.GetIntegrationAdapterRequest) (*integrationsv1.GetIntegrationAdapterResponse, error) {
	row, err := s.store.Integrations.GetIntegrationAdapter(ctx, req.AdapterId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "adapter %q not found", req.AdapterId)
		}
		return nil, status.Errorf(codes.Internal, "get adapter: %v", err)
	}
	adapter := convert.IntegrationAdapterFromDB(row)
	caps, err := s.store.Integrations.ListIntegrationAdapterCapabilities(ctx, row.AdapterID)
	if err == nil {
		for _, c := range caps {
			adapter.Capabilities = append(adapter.Capabilities, convert.AdapterCapabilityFromDB(c))
		}
	}
	return &integrationsv1.GetIntegrationAdapterResponse{Adapter: adapter}, nil
}

// SetIntegrationAdapterState enables or disables an adapter.
func (s *Service) SetIntegrationAdapterState(ctx context.Context, req *integrationsv1.SetIntegrationAdapterStateRequest) (*integrationsv1.SetIntegrationAdapterStateResponse, error) {
	row, err := s.store.Integrations.GetIntegrationAdapter(ctx, req.AdapterId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "adapter %q not found", req.AdapterId)
		}
		return nil, status.Errorf(codes.Internal, "get adapter: %v", err)
	}

	previousState := row.State
	var nextState int64
	if req.Enabled {
		nextState = int64(integrationsv1.AdapterState_ADAPTER_STATE_READY)
	} else {
		nextState = int64(integrationsv1.AdapterState_ADAPTER_STATE_DISABLED)
	}

	now := convert.NowDB()
	if err := s.store.Integrations.SetIntegrationAdapterState(ctx, integrationstore.SetIntegrationAdapterStateParams{
		AdapterID: req.AdapterId,
		State:     nextState,
		UpdatedAt: now,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "set adapter state: %v", err)
	}

	// Append state-change event.
	reason := req.Reason
	if err := s.store.Integrations.AppendIntegrationAdapterStateEvent(ctx, integrationstore.AppendIntegrationAdapterStateEventParams{
		AdapterID:     req.AdapterId,
		Sequence:      sequenceFromNow(),
		PreviousState: &previousState,
		NextState:     nextState,
		Reason:        nullableStr(reason),
		ChangedAt:     now,
	}); err != nil {
		// Non-fatal: event append failure should not block the state update.
		_ = fmt.Errorf("append state event: %w", err)
	}

	// Notify in-process registry so it immediately honours the new state.
	if s.notifier != nil {
		if req.Enabled {
			s.notifier.Enable(req.AdapterId)
		} else {
			s.notifier.Disable(req.AdapterId)
		}
	}

	// Return updated adapter.
	updated, err := s.store.Integrations.GetIntegrationAdapter(ctx, req.AdapterId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload adapter: %v", err)
	}
	adapter := convert.IntegrationAdapterFromDB(updated)
	caps, _ := s.store.Integrations.ListIntegrationAdapterCapabilities(ctx, updated.AdapterID)
	for _, c := range caps {
		adapter.Capabilities = append(adapter.Capabilities, convert.AdapterCapabilityFromDB(c))
	}
	return &integrationsv1.SetIntegrationAdapterStateResponse{Adapter: adapter}, nil
}

// ValidateIntegrationAdapter tests the live adapter connection.
func (s *Service) ValidateIntegrationAdapter(ctx context.Context, req *integrationsv1.ValidateIntegrationAdapterRequest) (*integrationsv1.ValidateIntegrationAdapterResponse, error) {
	if s.validator == nil {
		return nil, status.Error(codes.Unimplemented, "validator not configured")
	}
	ok, err := s.validator.Validate(ctx, req.AdapterId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "validate: %v", err)
	}
	return &integrationsv1.ValidateIntegrationAdapterResponse{Ok: ok}, nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func sequenceFromNow() int64 {
	return time.Now().UnixNano()
}
