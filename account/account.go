// Package account implements the subscription-related az account commands.
package account

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type Subscription = armsubscriptions.Subscription
type Location = armsubscriptions.Location

// Client reads ARM subscriptions in the authenticated tenant.
// Unlike az account list, List does not enumerate the CLI's cross-tenant cache.
type Client struct{ sdk *armsubscriptions.Client }

func NewClient(credential azcore.TokenCredential, options *arm.ClientOptions) (*Client, error) {
	sdk, err := armsubscriptions.NewClient(credential, options)
	if err != nil {
		return nil, err
	}
	return &Client{sdk: sdk}, nil
}

// List returns all subscriptions visible through ARM to the authenticated tenant.
func (c *Client) List(ctx context.Context) ([]*Subscription, error) {
	result, err := paging.Collect(ctx, c.sdk.NewListPager(nil), func(p armsubscriptions.ClientListResponse) []*Subscription { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("account list: %w", err)
	}
	return result, nil
}

// Show retrieves a subscription by its UUID.
func (c *Client) Show(ctx context.Context, subscriptionID string) (*Subscription, error) {
	if err := check.SubscriptionID(subscriptionID); err != nil {
		return nil, err
	}
	res, err := c.sdk.Get(ctx, subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("account show %q: %w", subscriptionID, err)
	}
	return &res.Subscription, nil
}

// ListLocations returns regions for the specified subscription (az account list-locations).
// A listed region does not guarantee support for a particular model or SKU.
func (c *Client) ListLocations(ctx context.Context, subscriptionID string) ([]*Location, error) {
	if err := check.SubscriptionID(subscriptionID); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListLocationsPager(subscriptionID, nil), func(p armsubscriptions.ClientListLocationsResponse) []*Location { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("account list-locations: %w", err)
	}
	return result, nil
}
