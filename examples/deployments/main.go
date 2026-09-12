// Deployments demonstrates the facade. It defaults to read-only list.
// All mutations require explicit -apply, model/SKU choices and a target name.
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
	"github.com/nuitsjp/azurego/cognitiveservices"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	action := flag.String("action", "list", "list, show, skus, create, update, delete")
	subscription := flag.String("subscription", "", "Required subscription UUID")
	tenant := flag.String("tenant", "", "Optional tenant ID")
	resourceGroup := flag.String("resource-group", "", "Required resource group")
	account := flag.String("account", "", "Required existing Foundry/OpenAI resource name")
	name := flag.String("deployment", "", "Deployment name (required except for list)")
	format := flag.String("model-format", "", "Model format from the model catalog")
	model := flag.String("model-name", "", "Model name from the model catalog")
	version := flag.String("model-version", "", "Explicit model version from the model catalog")
	source := flag.String("model-source", "", "Optional model source")
	sku := flag.String("sku-name", "", "Explicit deployment SKU, for example GlobalStandard")
	capacity := flag.Int64("sku-capacity", 0, "Explicit capacity in this model/SKU's unit")
	rai := flag.String("rai-policy", "", "Optional existing content-filter policy name")
	apply := flag.Bool("apply", false, "Explicitly authorize a create/update/delete operation")
	timeout := flag.Duration("timeout", 15*time.Minute, "Overall deadline including completion wait")
	flag.Parse()
	if *subscription == "" || *resourceGroup == "" || *account == "" {
		return fmt.Errorf("-subscription, -resource-group and -account are required")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	switch *action {
	case "list":
	case "show", "skus", "create", "update", "delete":
		if *name == "" {
			return fmt.Errorf("-deployment is required for %s", *action)
		}
	default:
		return fmt.Errorf("unknown action %q", *action)
	}
	writing := *action == "create" || *action == "update" || *action == "delete"
	if writing && !*apply {
		return fmt.Errorf("%s changes Azure resources; review the target and add -apply explicitly", *action)
	}
	if *action == "create" || *action == "update" {
		if *sku == "" || *capacity <= 0 || *capacity > (1<<31)-1 {
			return fmt.Errorf("a non-empty -sku-name and -sku-capacity between 1 and 2147483647 are required")
		}
	}
	if *action == "create" && (*format == "" || *model == "" || *version == "") {
		return fmt.Errorf("create requires -model-format, -model-name and -model-version")
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, *timeout)
	defer cancel()
	client, err := azurego.New(&azurego.Options{SubscriptionID: *subscription, TenantID: *tenant})
	if err != nil {
		return err
	}
	d := client.CognitiveServices.Account.Deployment
	var output any
	if writing {
		fmt.Fprintf(os.Stderr, "%s: subscription=%s resourceGroup=%s account=%s deployment=%s\n", *action, *subscription, *resourceGroup, *account, *name)
	}
	switch *action {
	case "list":
		output, err = d.List(ctx, *resourceGroup, *account)
	case "show":
		output, err = d.Show(ctx, *resourceGroup, *account, *name)
	case "skus":
		output, err = d.ListSKUs(ctx, *resourceGroup, *account, *name)
	case "create":
		output, err = d.Create(ctx, *resourceGroup, *account, *name, cognitiveservices.CreateOptions{
			ModelFormat: *format, ModelName: *model, ModelVersion: *version, ModelSource: *source,
			SKUName: *sku, SKUCapacity: int32(*capacity), RaiPolicyName: *rai,
		})
	case "update":
		output, err = d.Update(ctx, *resourceGroup, *account, *name, cognitiveservices.UpdateOptions{SKUName: *sku, SKUCapacity: int32(*capacity)})
	case "delete":
		err = d.Delete(ctx, *resourceGroup, *account, *name)
		if err == nil {
			output = map[string]string{"deleted": *name}
		}
	}
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
