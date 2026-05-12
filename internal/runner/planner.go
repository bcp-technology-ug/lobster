package runner

import (
	"context"
	"strings"
	"time"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	planstore "github.com/bcp-technology/lobster/gen/sqlc/plan"
	"github.com/bcp-technology/lobster/internal/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Plan parses feature files for the selector, builds an execution plan, and
// optionally persists it to the store.
func (p *Planner) Plan(ctx context.Context, req *planv1.PlanRequest) (*planv1.PlanResponse, error) {
	if req.Selector == nil {
		return nil, status.Error(codes.InvalidArgument, "selector is required")
	}

	cfg, err := p.configFn(ctx, req.Selector.WorkspaceId, req.Selector.ProfileName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve config: %v", err)
	}

	featurePaths := cfg.FeaturePaths
	if req.Selector.FeaturePath != "" {
		featurePaths = []string{req.Selector.FeaturePath}
	}

	var features []*parser.Feature
	for _, fp := range featurePaths {
		parsed, parseErr := parser.ParseGlob(fp)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "parse features %q: %v", fp, parseErr)
		}
		features = append(features, parsed...)
	}

	planID := newUUID()
	now := time.Now().UTC()

	var selectorFP *string
	if req.Selector.FeaturePath != "" {
		s := req.Selector.FeaturePath
		selectorFP = &s
	}
	var selectorTag *string
	if req.Selector.TagExpression != "" {
		s := req.Selector.TagExpression
		selectorTag = &s
	}
	var selectorProf *string
	if req.Selector.ProfileName != "" {
		s := req.Selector.ProfileName
		selectorProf = &s
	}

	plan := &planv1.ExecutionPlan{
		PlanId:      planID,
		WorkspaceId: req.Selector.WorkspaceId,
		ProfileName: req.Selector.ProfileName,
		CreatedAt:   timestamppb.New(now),
		Selector: &runv1.RunSelector{
			WorkspaceId:   req.Selector.WorkspaceId,
			FeaturePath:   req.Selector.FeaturePath,
			TagExpression: req.Selector.TagExpression,
			ProfileName:   req.Selector.ProfileName,
		},
	}

	for _, feature := range features {
		for _, scenario := range feature.Scenarios {
			if req.Selector.TagExpression != "" && !matchesTagExpression(scenario.Tags, req.Selector.TagExpression) {
				continue
			}
			exIdx := int32(-1)
			sp := &planv1.ScenarioPlan{
				ScenarioId:   scenario.DeterministicID,
				FeatureName:  feature.Name,
				ScenarioName: scenario.Name,
				Tags:         scenario.Tags,
				DeterministicKey: &commonv1.DeterministicScenarioKey{
					FeatureName:          feature.Name,
					ScenarioName:         scenario.Name,
					ExampleRowIndex:      exIdx,
					NormalizationVersion: "sha256-v1",
					StableHash:           scenario.DeterministicID,
				},
			}
			plan.Scenarios = append(plan.Scenarios, sp)
		}
	}

	if req.PersistArtifact && p.store != nil {
		if err := persistPlan(ctx, p.store.Plan, plan, selectorFP, selectorTag, selectorProf, now); err != nil {
			return nil, status.Errorf(codes.Internal, "persist plan: %v", err)
		}
	}

	return &planv1.PlanResponse{Plan: plan}, nil
}

func persistPlan(
	ctx context.Context,
	q *planstore.Queries,
	plan *planv1.ExecutionPlan,
	selectorFP, selectorTag, selectorProf *string,
	now time.Time,
) error {
	nowStr := now.Format(time.RFC3339Nano)
	if err := q.CreateExecutionPlan(ctx, planstore.CreateExecutionPlanParams{
		PlanID:                planID(plan),
		WorkspaceID:           plan.WorkspaceId,
		ProfileName:           plan.ProfileName,
		SelectorFeaturePath:   selectorFP,
		SelectorTagExpression: selectorTag,
		SelectorProfileName:   selectorProf,
		CreatedAt:             nowStr,
	}); err != nil {
		return err
	}
	for i, sc := range plan.Scenarios {
		normVersion := ""
		stableHash := ""
		featureName := sc.FeatureName
		scenarioName := sc.ScenarioName
		exRow := int64(-1)
		if sc.DeterministicKey != nil {
			normVersion = sc.DeterministicKey.NormalizationVersion
			stableHash = sc.DeterministicKey.StableHash
			exRow = int64(sc.DeterministicKey.ExampleRowIndex)
		}
		if err := q.UpsertExecutionPlanScenario(ctx, planstore.UpsertExecutionPlanScenarioParams{
			PlanID:                            plan.PlanId,
			ScenarioID:                        sc.ScenarioId,
			Ordinal:                           int64(i),
			FeatureName:                       featureName,
			ScenarioName:                      scenarioName,
			DeterministicFeatureName:          &featureName,
			DeterministicScenarioName:         &scenarioName,
			DeterministicExampleRowIndex:      &exRow,
			DeterministicNormalizationVersion: &normVersion,
			DeterministicStableHash:           &stableHash,
		}); err != nil {
			return err
		}
		for j, tag := range sc.Tags {
			_ = q.CreateExecutionPlanScenarioTag(ctx, planstore.CreateExecutionPlanScenarioTagParams{
				PlanID:     plan.PlanId,
				ScenarioID: sc.ScenarioId,
				Ordinal:    int64(j),
				Tag:        tag,
			})
		}
	}
	return nil
}

// planID is a thin helper to unwrap the plan ID proto field.
func planID(plan *planv1.ExecutionPlan) string { return plan.PlanId }

// matchesTagExpression returns true if the scenario tags satisfy the expression.
// v0.1 supports:
//
//	"@smoke"         → must have @smoke
//	"@smoke,@fast"   → must have @smoke OR @fast (comma = OR)
//	"@smoke @fast"   → must have @smoke AND @fast (space = AND)
//	"~@wip"          → must NOT have @wip
func matchesTagExpression(tags []string, expr string) bool {
	if expr == "" {
		return true
	}
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
		tagSet[strings.TrimPrefix(t, "@")] = true
	}

	for _, orTerm := range strings.Split(expr, ",") {
		andTerms := strings.Fields(strings.TrimSpace(orTerm))
		all := true
		for _, term := range andTerms {
			neg := strings.HasPrefix(term, "~")
			tag := strings.TrimPrefix(term, "~")
			tag = strings.TrimPrefix(tag, "@")
			has := tagSet[tag] || tagSet["@"+tag]
			if neg {
				has = !has
			}
			if !has {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}
