// Models lists available account/regional models and optional regional quota.
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
	subscription := flag.String("subscription", "", "Required subscription UUID")
	tenant := flag.String("tenant", "", "Optional tenant ID")
	resourceGroup := flag.String("resource-group", "", "Resource group for account-specific models")
	account := flag.String("account", "", "Existing Foundry/OpenAI resource name, not project name")
	location := flag.String("location", "", "Optional region for regional models and quota")
	timeout := flag.Duration("timeout", 2*time.Minute, "Overall deadline")
	flag.Parse()
	if *subscription == "" {
		return fmt.Errorf("-subscription is required")
	}
	if *account == "" && *location == "" {
		return fmt.Errorf("specify -account (with -resource-group), -location, or both")
	}
	if (*account == "") != (*resourceGroup == "") {
		return fmt.Errorf("-account and -resource-group must be specified together")
	}
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
	if *account != "" {
		output["accountModels"], err = client.CognitiveServices.Account.ListModels(ctx, *resourceGroup, *account)
		if err != nil {
			return err
		}
	}
	if *location != "" {
		output["regionalModels"], err = client.CognitiveServices.Model.List(ctx, *location)
		if err != nil {
			return err
		}
		output["quotaUsage"], err = client.CognitiveServices.Usage.List(ctx, *location)
		if err != nil {
			return err
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
