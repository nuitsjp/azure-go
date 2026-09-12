// Discover is a read-only example. It never chooses the first subscription.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/nuitsjp/azurego"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	subscription := flag.String("subscription", "", "Subscription UUID; omit to list subscriptions only")
	tenant := flag.String("tenant", "", "Tenant ID; defaults to the Azure CLI selection")
	timeout := flag.Duration("timeout", 2*time.Minute, "Overall deadline")
	flag.Parse()
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, *timeout)
	defer cancel()
	client, err := azurego.New(&azurego.Options{SubscriptionID: *subscription, TenantID: *tenant})
	if err != nil {
		return err
	}
	output := make(map[string]any)
	if *subscription == "" {
		output["subscriptions"], err = client.Account.List(ctx)
		if err != nil {
			return err
		}
	} else {
		output["subscription"], err = client.Account.Show(ctx, *subscription)
		if err != nil {
			return err
		}
		output["resourceGroups"], err = client.Group.List(ctx)
		if err != nil {
			return err
		}
		output["cognitiveServicesAccounts"], err = client.CognitiveServices.Account.List(ctx, "")
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
