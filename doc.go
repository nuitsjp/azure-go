// Package azurego provides a small Azure management facade with Azure CLI-style
// namespaces. Azure CLI is used only to obtain tokens; all resource operations
// use the official Azure SDK for Go and Azure Resource Manager directly.
//
// Clients do not sign in interactively or mutate Azure CLI configuration.
// Run az login before using the default credential. Construct a new Client for
// each subscription; client construction does not make network requests.
//
// List methods collect every page in memory. Mutating deployment methods wait
// for the Azure long-running operation to finish. Always pass an appropriate
// context deadline. Canceling the context stops waiting, not the Azure operation.
//
// Models returned by this package are Azure SDK model types, not CLI JSON DTOs.
package azurego
