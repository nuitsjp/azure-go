package cognitiveservices

import (
	"context"
	"fmt"

	sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type ModelClient struct{ sdk *sdk.ModelsClient }

// List implements az cognitiveservices model list --location. It reads the ARM
// regional catalog, not the entire Foundry portal/Marketplace model catalog.
func (c *ModelClient) List(ctx context.Context, location string) ([]*Model, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Required("location", location); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListPager(location, nil), func(p sdk.ModelsClientListResponse) []*Model { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices model list %q: %w", location, err)
	}
	return result, nil
}
