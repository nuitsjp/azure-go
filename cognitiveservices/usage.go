package cognitiveservices

import (
	"context"
	"fmt"

	sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type UsageClient struct{ sdk *sdk.UsagesClient }

// List implements az cognitiveservices usage list --location.
// Values describe quota usage, not token billing or actual model inference usage.
func (c *UsageClient) List(ctx context.Context, location string) ([]*Usage, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Required("location", location); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListPager(location, nil), func(p sdk.UsagesClientListResponse) []*Usage { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices usage list %q: %w", location, err)
	}
	return result, nil
}
