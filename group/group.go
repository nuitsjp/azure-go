// Package group implements the read-only az group commands.
package group

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type ResourceGroup = armresources.ResourceGroup
type Client struct {
	sdk *armresources.ResourceGroupsClient
}

// NewClient accepts an empty subscription for discovery-only root clients.
// In that case its operations return ErrSubscriptionRequired.
func NewClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*Client, error) {
	if subscriptionID == "" {
		return &Client{}, nil
	}
	if err := check.SubscriptionID(subscriptionID); err != nil {
		return nil, err
	}
	sdk, err := armresources.NewResourceGroupsClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &Client{sdk: sdk}, nil
}

func (c *Client) List(ctx context.Context) ([]*ResourceGroup, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListPager(nil), func(p armresources.ResourceGroupsClientListResponse) []*ResourceGroup { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("group list: %w", err)
	}
	return result, nil
}

func (c *Client) Show(ctx context.Context, name string) (*ResourceGroup, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Required("resourceGroup", name); err != nil {
		return nil, err
	}
	res, err := c.sdk.Get(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("group show %q: %w", name, err)
	}
	return &res.ResourceGroup, nil
}
