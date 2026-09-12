package azurego

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
)

// Options configures a client. A nil Options is valid for subscription discovery.
type Options struct {
	// SubscriptionID is the ARM subscription UUID, not the display name.
	// When empty, only Account operations are available. Other operations return
	// ErrSubscriptionRequired, rather than selecting a subscription implicitly.
	SubscriptionID string
	// TenantID pins the CLI credential to a tenant. Empty uses CLI defaults.
	TenantID string
	// Credential optionally replaces AzureCLICredential, primarily for tests or
	// applications that already have a credential. TenantID must then be empty;
	// authentication tenant selection is the supplied credential's responsibility.
	Credential azcore.TokenCredential
	// ARMOptions configures the official SDK transport, retries and cloud.
	// Automatic resource-provider registration is always disabled by AzureGo.
	ARMOptions *arm.ClientOptions
}
