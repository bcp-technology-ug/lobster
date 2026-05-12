package convert

import (
	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
)

// StackFromDB converts a sqlc Stack row to the proto Stack message.
// Components are attached by the service layer.
func StackFromDB(s stackstore.Stack) *stackv1.Stack {
	return &stackv1.Stack{
		StackId:     s.StackID,
		WorkspaceId: s.WorkspaceID,
		ProjectName: s.ProjectName,
		Status:      stackv1.StackStatus(s.Status),
		CreatedAt:   TimestampFromDBStr(s.CreatedAt),
		UpdatedAt:   TimestampFromDBStr(s.UpdatedAt),
	}
}

// StackComponentFromDB converts a sqlc StackComponent row to proto.
func StackComponentFromDB(c stackstore.StackComponent) *stackv1.StackComponent {
	sc := &stackv1.StackComponent{
		Name:      c.Name,
		Health:    stackv1.ServiceHealth(c.Health),
		UpdatedAt: TimestampFromDBStr(c.UpdatedAt),
	}
	if c.Image != nil {
		sc.Image = *c.Image
	}
	if c.ContainerID != nil {
		sc.ContainerId = *c.ContainerID
	}
	if c.Status != nil {
		sc.Status = *c.Status
	}
	return sc
}
