// Package check contains small, SDK-independent input checks.
package check

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrSubscriptionRequired = errors.New("azurego: SubscriptionID is required for this operation")
var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func SubscriptionID(value string) error {
	if value == "" {
		return ErrSubscriptionRequired
	}
	if !uuid.MatchString(value) {
		return fmt.Errorf("azurego: SubscriptionID must be a UUID, not a subscription display name")
	}
	return nil
}

// Scope guards unconfigured namespace clients before they dereference the SDK.
func Scope(configured bool) error {
	if !configured {
		return ErrSubscriptionRequired
	}
	return nil
}

func Required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("azurego: %s is required", name)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("azurego: %s must not have surrounding whitespace", name)
	}
	return nil
}

func Resource(resourceGroup, account string) error {
	if err := Required("resourceGroup", resourceGroup); err != nil {
		return err
	}
	return Required("accountName", account)
}

func Deployment(resourceGroup, account, deployment string) error {
	if err := Resource(resourceGroup, account); err != nil {
		return err
	}
	return Required("deploymentName", deployment)
}
