package azurego

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/nuitsjp/azurego/account"
	"github.com/nuitsjp/azurego/cognitiveservices"
	"github.com/nuitsjp/azurego/group"
	"github.com/nuitsjp/azurego/internal/check"
)

// ErrSubscriptionRequired indicates that a subscription-scoped operation was
// called on a client created without Options.SubscriptionID. Use errors.Is.
var ErrSubscriptionRequired = check.ErrSubscriptionRequired

// Client groups operations by their Azure CLI command group.
// Construct with New. Treat the exported namespace fields as read-only.
type Client struct {
	Account           *account.Client
	Group             *group.Client
	CognitiveServices *cognitiveservices.Client
}

// New constructs clients without authenticating or making network requests.
// It never invokes az login or az account set. Subscription selection is explicit.
func New(options *Options) (*Client, error) {
	opts := Options{}
	if options != nil {
		opts = *options
	}
	if opts.SubscriptionID != "" {
		if err := check.SubscriptionID(opts.SubscriptionID); err != nil {
			return nil, err
		}
	}
	cred := opts.Credential
	if cred != nil && opts.TenantID != "" {
		return nil, fmt.Errorf("azurego: TenantID cannot be combined with Credential; configure the supplied credential instead")
	}
	if cred == nil {
		var err error
		cred, err = azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
			Subscription: opts.SubscriptionID,
			TenantID:     opts.TenantID,
		})
		if err != nil {
			return nil, fmt.Errorf("azurego: create CLI credential: %w", err)
		}
	}
	armOpts := arm.ClientOptions{}
	if opts.ARMOptions != nil {
		armOpts = *opts.ARMOptions
	}
	// Reading resources must not silently register a resource provider.
	armOpts.DisableRPRegistration = true
	a, err := account.NewClient(cred, &armOpts)
	if err != nil {
		return nil, fmt.Errorf("azurego: initialize account: %w", err)
	}
	g, err := group.NewClient(opts.SubscriptionID, cred, &armOpts)
	if err != nil {
		return nil, fmt.Errorf("azurego: initialize group: %w", err)
	}
	cs, err := cognitiveservices.NewClient(opts.SubscriptionID, cred, &armOpts)
	if err != nil {
		return nil, fmt.Errorf("azurego: initialize cognitiveservices: %w", err)
	}
	return &Client{Account: a, Group: g, CognitiveServices: cs}, nil
}
