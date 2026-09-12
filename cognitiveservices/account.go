package cognitiveservices

import (
	"context"
	"fmt"

	sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type AccountClient struct {
	Deployment *DeploymentClient
	sdk        *sdk.AccountsClient
}

// List returns Cognitive Services accounts. An empty resourceGroup means the
// entire selected subscription. No kind filter is applied; inspect Kind to find
// AIServices (Foundry) and OpenAI accounts.
func (c *AccountClient) List(ctx context.Context, resourceGroup string) ([]*Account, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	var result []*Account
	var err error
	if resourceGroup == "" {
		result, err = paging.Collect(ctx, c.sdk.NewListPager(nil), func(p sdk.AccountsClientListResponse) []*Account { return p.Value })
	} else {
		if err := check.Required("resourceGroup", resourceGroup); err != nil {
			return nil, err
		}
		result, err = paging.Collect(ctx, c.sdk.NewListByResourceGroupPager(resourceGroup, nil), func(p sdk.AccountsClientListByResourceGroupResponse) []*Account { return p.Value })
	}
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account list: %w", err)
	}
	return result, nil
}

func (c *AccountClient) Show(ctx context.Context, resourceGroup, name string) (*Account, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Resource(resourceGroup, name); err != nil {
		return nil, err
	}
	res, err := c.sdk.Get(ctx, resourceGroup, name, nil)
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account show %q: %w", name, err)
	}
	return &res.Account, nil
}

// ListModels reads the models exposed for an existing account, including model
// format, version and SKU metadata. Presence is not a quota/capacity guarantee.
func (c *AccountClient) ListModels(ctx context.Context, resourceGroup, name string) ([]*AccountModel, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Resource(resourceGroup, name); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListModelsPager(resourceGroup, name, nil), func(p sdk.AccountsClientListModelsResponse) []*AccountModel { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account list-models %q: %w", name, err)
	}
	return result, nil
}
