package cognitiveservices

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/nuitsjp/azurego/internal/check"
	"github.com/nuitsjp/azurego/internal/paging"
)

type DeploymentClient struct{ sdk *sdk.DeploymentsClient }

// CreateOptions mirrors common az cognitiveservices account deployment create
// flags. All model and SKU choices are explicit; there are no paid defaults.
type CreateOptions struct {
	ModelFormat  string
	ModelName    string
	ModelVersion string
	// ModelSource optionally identifies a model source for supported formats.
	ModelSource string
	SKUName     string
	// SKUCapacity is the provider's capacity unit, NOT universally tokens/minute.
	SKUCapacity int32
	// RaiPolicyName selects an already-existing content-filter policy.
	RaiPolicyName string
	// VersionUpgradeOption optionally selects the provider's upgrade policy.
	VersionUpgradeOption *sdk.DeploymentModelVersionUpgradeOption
}

// UpdateOptions changes SKU/capacity only, using PATCH rather than re-submitting
// model properties. Version changes use Create with a full desired definition.
// Update is an AzureGo extension; CLI's deployment group has no update command.
type UpdateOptions struct {
	SKUName     string
	SKUCapacity int32
}

func (o CreateOptions) validate() error {
	for _, p := range []struct{ name, value string }{
		{"ModelFormat", o.ModelFormat}, {"ModelName", o.ModelName},
		{"ModelVersion", o.ModelVersion}, {"SKUName", o.SKUName},
	} {
		if err := check.Required(p.name, p.value); err != nil {
			return err
		}
	}
	if o.SKUCapacity <= 0 {
		return fmt.Errorf("azurego: SKUCapacity must be greater than zero")
	}
	if o.ModelSource != "" {
		if err := check.Required("ModelSource", o.ModelSource); err != nil {
			return err
		}
	}
	if o.RaiPolicyName != "" {
		if err := check.Required("RaiPolicyName", o.RaiPolicyName); err != nil {
			return err
		}
	}
	return nil
}

func (c *DeploymentClient) List(ctx context.Context, resourceGroup, accountName string) ([]*Deployment, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Resource(resourceGroup, accountName); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListPager(resourceGroup, accountName, nil), func(p sdk.DeploymentsClientListResponse) []*Deployment { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account deployment list: %w", err)
	}
	return result, nil
}

func (c *DeploymentClient) Show(ctx context.Context, resourceGroup, accountName, name string) (*Deployment, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Deployment(resourceGroup, accountName, name); err != nil {
		return nil, err
	}
	res, err := c.sdk.Get(ctx, resourceGroup, accountName, name, nil)
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account deployment show %q: %w", name, err)
	}
	return &res.Deployment, nil
}

// Create creates OR updates an account-level deployment via ARM PUT, like the
// CLI create command. An existing deployment with the same name can be changed.
// This is not a partial update: supply the complete intended settings exposed
// here. Existing advanced properties not represented by CreateOptions are not
// round-tripped. For capacity-only edits, prefer Update.
//
// The call waits for completion. Cancellation stops waiting, not the server-side
// operation. Creation, updates and subsequent inference may incur Azure charges.
func (c *DeploymentClient) Create(ctx context.Context, resourceGroup, accountName, name string, options CreateOptions) (*Deployment, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Deployment(resourceGroup, accountName, name); err != nil {
		return nil, err
	}
	if err := options.validate(); err != nil {
		return nil, err
	}
	model := &sdk.DeploymentModel{
		Format: to.Ptr(options.ModelFormat), Name: to.Ptr(options.ModelName), Version: to.Ptr(options.ModelVersion),
	}
	if options.ModelSource != "" {
		model.Source = to.Ptr(options.ModelSource)
	}
	properties := &sdk.DeploymentProperties{Model: model, VersionUpgradeOption: options.VersionUpgradeOption}
	if options.RaiPolicyName != "" {
		properties.RaiPolicyName = to.Ptr(options.RaiPolicyName)
	}
	desired := sdk.Deployment{
		SKU:        &sdk.SKU{Name: to.Ptr(options.SKUName), Capacity: to.Ptr(options.SKUCapacity)},
		Properties: properties,
	}
	poller, err := c.sdk.BeginCreateOrUpdate(ctx, resourceGroup, accountName, name, desired, nil)
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account deployment create %q: %w", name, err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("wait for deployment create %q (operation may still be running): %w", name, err)
	}
	return &res.Deployment, nil
}

// Update patches the SKU and capacity of an existing deployment and waits.
// It does not implement model-version or content-filter changes.
func (c *DeploymentClient) Update(ctx context.Context, resourceGroup, accountName, name string, options UpdateOptions) (*Deployment, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Deployment(resourceGroup, accountName, name); err != nil {
		return nil, err
	}
	if err := check.Required("SKUName", options.SKUName); err != nil {
		return nil, err
	}
	if options.SKUCapacity <= 0 {
		return nil, fmt.Errorf("azurego: SKUCapacity must be greater than zero")
	}
	patch := sdk.PatchResourceTagsAndSKU{SKU: &sdk.SKU{Name: to.Ptr(options.SKUName), Capacity: to.Ptr(options.SKUCapacity)}}
	poller, err := c.sdk.BeginUpdate(ctx, resourceGroup, accountName, name, patch, nil)
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account deployment update %q: %w", name, err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("wait for deployment update %q (operation may still be running): %w", name, err)
	}
	return &res.Deployment, nil
}

// Delete removes the named deployment and waits for completion. There is no
// confirmation prompt inside the library. The caller must authorize the deletion.
// A missing deployment follows the service's semantics; AzureGo does not hide errors.
func (c *DeploymentClient) Delete(ctx context.Context, resourceGroup, accountName, name string) error {
	if err := check.Scope(c.sdk != nil); err != nil {
		return err
	}
	if err := check.Deployment(resourceGroup, accountName, name); err != nil {
		return err
	}
	poller, err := c.sdk.BeginDelete(ctx, resourceGroup, accountName, name, nil)
	if err != nil {
		return fmt.Errorf("cognitiveservices account deployment delete %q: %w", name, err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("wait for deployment delete %q (operation may still be running): %w", name, err)
	}
	return nil
}

// ListSKUs retrieves the service's SKU/capacity rules for an existing deployment.
func (c *DeploymentClient) ListSKUs(ctx context.Context, resourceGroup, accountName, name string) ([]*SKUResource, error) {
	if err := check.Scope(c.sdk != nil); err != nil {
		return nil, err
	}
	if err := check.Deployment(resourceGroup, accountName, name); err != nil {
		return nil, err
	}
	result, err := paging.Collect(ctx, c.sdk.NewListSKUsPager(resourceGroup, accountName, name, nil), func(p sdk.DeploymentsClientListSKUsResponse) []*SKUResource { return p.Value })
	if err != nil {
		return nil, fmt.Errorf("cognitiveservices account deployment list-skus %q: %w", name, err)
	}
	return result, nil
}
