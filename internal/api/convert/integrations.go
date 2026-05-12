package convert

import (
	integrationsv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/integrations"
	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"
	"google.golang.org/protobuf/types/known/anypb"
)

// IntegrationAdapterFromDB converts a sqlc IntegrationAdapter row to proto.
// Capabilities are attached by the service layer.
func IntegrationAdapterFromDB(a integrationstore.IntegrationAdapter) *integrationsv1.IntegrationAdapter {
	adapter := &integrationsv1.IntegrationAdapter{
		AdapterId: a.AdapterID,
		Name:      a.Name,
		Type:      a.Type,
		State:     integrationsv1.AdapterState(a.State),
		UpdatedAt: TimestampFromDBStr(a.UpdatedAt),
	}
	if a.ConfigExtensionTypeUrl != nil && len(a.ConfigExtensionValue) > 0 {
		adapter.ConfigExtension = &anypb.Any{
			TypeUrl: *a.ConfigExtensionTypeUrl,
			Value:   a.ConfigExtensionValue,
		}
	}
	if a.StateExtensionTypeUrl != nil && len(a.StateExtensionValue) > 0 {
		adapter.StateExtension = &anypb.Any{
			TypeUrl: *a.StateExtensionTypeUrl,
			Value:   a.StateExtensionValue,
		}
	}
	return adapter
}

// AdapterCapabilityFromDB converts a capability row to proto.
func AdapterCapabilityFromDB(c integrationstore.IntegrationAdapterCapability) *integrationsv1.AdapterCapability {
	return &integrationsv1.AdapterCapability{
		Name:    c.Name,
		Enabled: c.Enabled == 1,
	}
}
