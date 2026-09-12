// Package cognitiveservices exposes the small subset of az cognitiveservices
// needed to discover Foundry resources and manage account-level deployments.
// It does not implement Azure Machine Learning workspace deployments, inference,
// fine-tuning jobs, or Foundry project management.
package cognitiveservices

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdk "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/nuitsjp/azurego/internal/check"
)

// SDK models are reused to preserve fields without a second model hierarchy.
type Account = sdk.Account
type AccountModel = sdk.AccountModel
type Model = sdk.Model
type Deployment = sdk.Deployment
type Usage = sdk.Usage
type SKUResource = sdk.SKUResource

type Client struct {
	Account *AccountClient
	Model   *ModelClient
	Usage   *UsageClient
}

func NewClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*Client, error) {
	c := &Client{
		Account: &AccountClient{Deployment: &DeploymentClient{}},
		Model:   &ModelClient{},
		Usage:   &UsageClient{},
	}
	if subscriptionID == "" {
		return c, nil
	}
	if err := check.SubscriptionID(subscriptionID); err != nil {
		return nil, err
	}
	f, err := sdk.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	c.Account.sdk = f.NewAccountsClient()
	c.Account.Deployment.sdk = f.NewDeploymentsClient()
	c.Model.sdk = f.NewModelsClient()
	c.Usage.sdk = f.NewUsagesClient()
	return c, nil
}
