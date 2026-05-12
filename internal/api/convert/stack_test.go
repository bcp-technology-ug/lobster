package convert

import (
	"testing"

	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
)

func TestStackFromDB_basic(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := stackstore.Stack{
		StackID:     "stack-001",
		WorkspaceID: "ws-001",
		ProfileName: "default",
		ProjectName: "my-project",
		Status:      int64(stackv1.StackStatus_STACK_STATUS_HEALTHY),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s := StackFromDB(row)
	if s.StackId != "stack-001" {
		t.Errorf("StackId: got %q want %q", s.StackId, "stack-001")
	}
	if s.WorkspaceId != "ws-001" {
		t.Errorf("WorkspaceId: got %q want %q", s.WorkspaceId, "ws-001")
	}
	if s.ProjectName != "my-project" {
		t.Errorf("ProjectName: got %q want %q", s.ProjectName, "my-project")
	}
	if s.Status != stackv1.StackStatus_STACK_STATUS_HEALTHY {
		t.Errorf("Status: got %v want HEALTHY", s.Status)
	}
	if s.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
	if s.UpdatedAt == nil {
		t.Error("UpdatedAt should not be nil")
	}
}

func TestStackComponentFromDB_basic(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := stackstore.StackComponent{
		StackID:   "stack-001",
		Name:      "api",
		Health:    int64(stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY),
		UpdatedAt: now,
	}
	sc := StackComponentFromDB(row)
	if sc.Name != "api" {
		t.Errorf("Name: got %q want %q", sc.Name, "api")
	}
	if sc.Health != stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY {
		t.Errorf("Health: got %v want HEALTHY", sc.Health)
	}
}

func TestStackComponentFromDB_withOptionalFields(t *testing.T) {
	t.Parallel()
	now := NowDB()
	image := "nginx:1.25"
	cid := "abc123def456"
	statusStr := "running"
	row := stackstore.StackComponent{
		StackID:     "stack-001",
		Name:        "nginx",
		Image:       &image,
		ContainerID: &cid,
		Status:      &statusStr,
		Health:      int64(stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY),
		UpdatedAt:   now,
	}
	sc := StackComponentFromDB(row)
	if sc.Image != "nginx:1.25" {
		t.Errorf("Image: got %q want %q", sc.Image, "nginx:1.25")
	}
	if sc.ContainerId != "abc123def456" {
		t.Errorf("ContainerId: got %q want %q", sc.ContainerId, "abc123def456")
	}
	if sc.Status != "running" {
		t.Errorf("Status: got %q want %q", sc.Status, "running")
	}
}
