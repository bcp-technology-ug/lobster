package convert

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
)

// RunFromDB converts a sqlc Run row to the proto Run message.
// Scenario results, hook results, and variables are loaded separately by the
// service layer and attached after this conversion.
func RunFromDB(r runstore.Run) *runv1.Run {
	run := &runv1.Run{
		RunId:       r.RunID,
		WorkspaceId: r.WorkspaceID,
		ProfileName: r.ProfileName,
		Status:      commonv1.RunStatus(r.Status),
		Summary: &runv1.RunSummary{
			TotalScenarios:   uint32(r.SummaryTotalScenarios),
			PassedScenarios:  uint32(r.SummaryPassedScenarios),
			FailedScenarios:  uint32(r.SummaryFailedScenarios),
			SkippedScenarios: uint32(r.SummarySkippedScenarios),
			Duration:         DurationFromNanos(r.SummaryDurationNanos),
		},
		CreatedAt: TimestampFromDBStr(r.CreatedAt),
		StartedAt: TimestampFromDB(r.StartedAt),
		EndedAt:   TimestampFromDB(r.EndedAt),
	}
	if r.FeatureName != nil || r.FeatureDescription != nil {
		run.Feature = &runv1.Feature{}
		if r.FeatureName != nil {
			run.Feature.Name = *r.FeatureName
		}
		if r.FeatureDescription != nil {
			run.Feature.Description = *r.FeatureDescription
		}
	}
	return run
}

// RunEventFromDB converts a stored run_event row into the proto RunEvent.
func RunEventFromDB(e runstore.RunEvent) *runv1.RunEvent {
	evt := &runv1.RunEvent{
		Sequence:   uint64(e.Sequence),
		RunId:      e.RunID,
		ObservedAt: TimestampFromDBStr(e.ObservedAt),
		EventType:  runv1.RunEventType(e.EventType),
		Terminal:   e.Terminal == 1,
	}

	switch runv1.RunEventType(e.EventType) {
	case runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS:
		if e.PayloadRunStatus != nil {
			evt.Payload = &runv1.RunEvent_RunStatus{
				RunStatus: commonv1.RunStatus(*e.PayloadRunStatus),
			}
		}
	case runv1.RunEventType_RUN_EVENT_TYPE_SUMMARY:
		if e.PayloadSummaryTotalScenarios != nil {
			var durationNs int64
			if e.PayloadSummaryDurationNanos != nil {
				durationNs = *e.PayloadSummaryDurationNanos
			}
			evt.Payload = &runv1.RunEvent_Summary{
				Summary: &runv1.RunSummary{
					TotalScenarios:   uint32(ptrInt64Val(e.PayloadSummaryTotalScenarios)),
					PassedScenarios:  uint32(ptrInt64Val(e.PayloadSummaryPassedScenarios)),
					FailedScenarios:  uint32(ptrInt64Val(e.PayloadSummaryFailedScenarios)),
					SkippedScenarios: uint32(ptrInt64Val(e.PayloadSummarySkippedScenarios)),
					Duration:         durationpb.New(time.Duration(durationNs)),
				},
			}
		}
	}
	// ScenarioResult, StepResult, HookResult payloads are assembled by the
	// service layer from supplementary tables when full detail is needed.
	// StreamRunEvents returns lightweight event frames directly from run_events.
	return evt
}

func ptrInt64Val(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
