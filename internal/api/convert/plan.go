package convert

import (
	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	planstore "github.com/bcp-technology/lobster/gen/sqlc/plan"
)

// ExecutionPlanFromDB converts a sqlc ExecutionPlan row to the proto message.
// Scenarios and artifact are attached by the service layer.
func ExecutionPlanFromDB(p planstore.ExecutionPlan) *planv1.ExecutionPlan {
	plan := &planv1.ExecutionPlan{
		PlanId:            p.PlanID,
		WorkspaceId:       p.WorkspaceID,
		ProfileName:       p.ProfileName,
		EstimatedDuration: DurationFromNanos(p.EstimatedDurationNanos),
		CreatedAt:         TimestampFromDBStr(p.CreatedAt),
	}
	if p.SelectorFeaturePath != nil || p.SelectorTagExpression != nil || p.SelectorProfileName != nil {
		plan.Selector = &runv1.RunSelector{
			WorkspaceId: p.WorkspaceID,
		}
		if p.SelectorFeaturePath != nil {
			plan.Selector.FeaturePath = *p.SelectorFeaturePath
		}
		if p.SelectorTagExpression != nil {
			plan.Selector.TagExpression = *p.SelectorTagExpression
		}
		if p.SelectorProfileName != nil {
			plan.Selector.ProfileName = *p.SelectorProfileName
		}
	}
	return plan
}

// ScenarioPlanFromDB converts a stored plan scenario row to proto.
func ScenarioPlanFromDB(s planstore.ExecutionPlanScenario) *planv1.ScenarioPlan {
	sp := &planv1.ScenarioPlan{
		ScenarioId:        s.ScenarioID,
		FeatureName:       s.FeatureName,
		ScenarioName:      s.ScenarioName,
		EstimatedDuration: DurationFromNanos(s.EstimatedDurationNanos),
	}
	if s.DeterministicFeatureName != nil || s.DeterministicScenarioName != nil || s.DeterministicStableHash != nil {
		sp.DeterministicKey = &commonv1.DeterministicScenarioKey{}
		if s.DeterministicFeatureName != nil {
			sp.DeterministicKey.FeatureName = *s.DeterministicFeatureName
		}
		if s.DeterministicScenarioName != nil {
			sp.DeterministicKey.ScenarioName = *s.DeterministicScenarioName
		}
		if s.DeterministicExampleRowIndex != nil {
			sp.DeterministicKey.ExampleRowIndex = int32(*s.DeterministicExampleRowIndex)
		}
		if s.DeterministicNormalizationVersion != nil {
			sp.DeterministicKey.NormalizationVersion = *s.DeterministicNormalizationVersion
		}
		if s.DeterministicStableHash != nil {
			sp.DeterministicKey.StableHash = *s.DeterministicStableHash
		}
	}
	return sp
}

// PlanArtifactFromDB converts a stored PlanArtifact row to proto.
func PlanArtifactFromDB(a planstore.PlanArtifact) *planv1.PlanArtifact {
	pa := &planv1.PlanArtifact{
		ArtifactId:  a.ArtifactID,
		StoragePath: a.StoragePath,
	}
	// Build envelope if any envelope fields are present.
	if a.EnvelopeMediaType != nil || a.EnvelopeCreatedAt != nil || a.EnvelopePayloadSha256 != nil {
		env := &commonv1.ArtifactEnvelope{}
		if a.EnvelopeSchemaVersion != nil {
			env.SchemaVersion = *a.EnvelopeSchemaVersion
		}
		if a.EnvelopeSchemaRevision != nil {
			env.SchemaRevision = uint32(*a.EnvelopeSchemaRevision)
		}
		if a.EnvelopeMediaType != nil {
			env.MediaType = *a.EnvelopeMediaType
		}
		if a.EnvelopeJsonExport != nil {
			env.JsonExport = *a.EnvelopeJsonExport
		}
		if a.EnvelopeCreatedAt != nil {
			env.CreatedAt = TimestampFromDB(a.EnvelopeCreatedAt)
		}
		if a.EnvelopePayloadSha256 != nil {
			env.PayloadSha256 = *a.EnvelopePayloadSha256
		}
		if a.EnvelopeCompressionType != nil {
			env.Compression = commonv1.CompressionType(*a.EnvelopeCompressionType)
		}
		if a.EnvelopeSignature != nil {
			env.Signature = *a.EnvelopeSignature
		}
		pa.Envelope = env
	}
	return pa
}
