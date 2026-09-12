package azurego_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/nuitsjp/azurego"
	cs "github.com/nuitsjp/azurego/cognitiveservices"
)

const sub = "00000000-0000-0000-0000-000000000001"
const rgPath = "/subscriptions/" + sub + "/resourceGroups/rg-test"
const accountPath = rgPath + "/providers/Microsoft.CognitiveServices/accounts/ai-test"
const deploymentPath = accountPath + "/deployments/d-test"
const deploymentBody = `{"id":"` + deploymentPath + `","name":"d-test","properties":{"provisioningState":"Succeeded","model":{"format":"OpenAI","name":"test-model","version":"test-version"}},"sku":{"name":"GlobalStandard","capacity":1}}`

type fakeCredential struct {
	calls atomic.Int32
	cause error
}

func (f *fakeCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	f.calls.Add(1)
	if err := ctx.Err(); err != nil {
		return azcore.AccessToken{}, err
	}
	if f.cause != nil {
		return azcore.AccessToken{}, f.cause
	}
	return azcore.AccessToken{Token: "local-unit-test-not-a-real-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func response(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

type step struct {
	method, path string
	status       int
	body         string
	inspect      func(*http.Request)
}

func clientWithSteps(t *testing.T, steps ...step) *azurego.Client {
	t.Helper()
	var mu sync.Mutex
	count := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if count >= len(steps) {
			return nil, fmt.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		s := steps[count]
		count++
		if r.Method != s.method || !strings.EqualFold(r.URL.Path, s.path) {
			t.Errorf("request %d: got %s %s; want %s %s", count, r.Method, r.URL.Path, s.method, s.path)
		}
		if r.URL.Host != "management.azure.com" {
			t.Errorf("unexpected host: %s", r.URL.Host)
		}
		if r.URL.Query().Get("api-version") == "" {
			t.Error("missing api-version")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing bearer token")
		}
		if s.inspect != nil {
			s.inspect(r)
		}
		return response(r, s.status, s.body), nil
	})
	c := newTestClient(t, sub, &fakeCredential{}, transport)
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		if count != len(steps) {
			t.Errorf("made %d requests; expected %d", count, len(steps))
		}
	})
	return c
}
func newTestClient(t *testing.T, subscription string, cred azcore.TokenCredential, transport policy.Transporter) *azurego.Client {
	t.Helper()
	c, err := azurego.New(&azurego.Options{
		SubscriptionID: subscription, Credential: cred,
		ARMOptions: &arm.ClientOptions{ClientOptions: policy.ClientOptions{Transport: transport, Retry: policy.RetryOptions{MaxRetries: -1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func mustJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestNewDoesNotAuthenticate(t *testing.T) {
	credential := &fakeCredential{}
	c := newTestClient(t, "", credential, transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("unexpected HTTP") }))
	if c.Account == nil || c.Group == nil || c.CognitiveServices.Account.Deployment == nil {
		t.Fatal("missing namespace")
	}
	if credential.calls.Load() != 0 {
		t.Fatal("constructor authenticated")
	}
	if _, err := c.Group.List(context.Background()); !errors.Is(err, azurego.ErrSubscriptionRequired) {
		t.Fatalf("got %v", err)
	}
	if _, err := c.CognitiveServices.Model.List(context.Background(), "eastus"); !errors.Is(err, azurego.ErrSubscriptionRequired) {
		t.Fatalf("got %v", err)
	}
	if _, err := c.CognitiveServices.Account.Deployment.List(context.Background(), "rg", "a"); !errors.Is(err, azurego.ErrSubscriptionRequired) {
		t.Fatalf("got %v", err)
	}
	if credential.calls.Load() != 0 {
		t.Fatal("invalid operation authenticated")
	}
}
func TestNewRejectsInvalidConfiguration(t *testing.T) {
	if _, err := azurego.New(&azurego.Options{SubscriptionID: "display-name"}); err == nil {
		t.Fatal("accepted display name")
	}
	if _, err := azurego.New(&azurego.Options{Credential: &fakeCredential{}, TenantID: sub}); err == nil {
		t.Fatal("accepted ambiguous credential config")
	}
}

func TestAccountListFollowsNextLink(t *testing.T) {
	next := "https://management.azure.com/subscriptions?api-version=2022-12-01&$skiptoken=page2"
	c := clientWithSteps(t,
		step{"GET", "/subscriptions", 200, `{"value":[{"subscriptionId":"` + sub + `","displayName":"one"}],"nextLink":"` + next + `"}`, nil},
		step{"GET", "/subscriptions", 200, `{"value":[{"subscriptionId":"00000000-0000-0000-0000-000000000002","displayName":"two"}]}`, func(r *http.Request) {
			if r.URL.Query().Get("$skiptoken") != "page2" {
				t.Error("lost continuation")
			}
		}},
	)
	got, err := c.Account.List(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("length=%d err=%v", len(got), err)
	}
}
func TestAccountShowAndLocations(t *testing.T) {
	c := clientWithSteps(t,
		step{"GET", "/subscriptions/" + sub, 200, `{"subscriptionId":"` + sub + `","displayName":"demo"}`, nil},
		step{"GET", "/subscriptions/" + sub + "/locations", 200, `{"value":[{"name":"eastus"}]}`, nil},
	)
	s, err := c.Account.Show(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	if s.DisplayName == nil || *s.DisplayName != "demo" {
		t.Fatalf("unexpected subscription: %+v", s)
	}
	locations, err := c.Account.ListLocations(context.Background(), sub)
	if err != nil || len(locations) != 1 {
		t.Fatalf("locations=%v err=%v", locations, err)
	}
}
func TestGroupListAndShow(t *testing.T) {
	c := clientWithSteps(t,
		step{"GET", "/subscriptions/" + sub + "/resourcegroups", 200, `{"value":[{"name":"rg-test","location":"eastus"}]}`, nil},
		step{"GET", rgPath, 200, `{"name":"rg-test","location":"eastus"}`, nil},
	)
	got, err := c.Group.List(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("groups=%v err=%v", got, err)
	}
	one, err := c.Group.Show(context.Background(), "rg-test")
	if err != nil || one.Name == nil || *one.Name != "rg-test" {
		t.Fatalf("group=%v err=%v", one, err)
	}
}
func TestCognitiveAccountDiscovery(t *testing.T) {
	c := clientWithSteps(t,
		step{"GET", "/subscriptions/" + sub + "/providers/Microsoft.CognitiveServices/accounts", 200, `{"value":[{"name":"ai-test","kind":"AIServices"}]}`, nil},
		step{"GET", rgPath + "/providers/Microsoft.CognitiveServices/accounts", 200, `{"value":[{"name":"ai-test","kind":"AIServices"}]}`, nil},
		step{"GET", accountPath, 200, `{"name":"ai-test","kind":"AIServices"}`, nil},
		step{"GET", accountPath + "/models", 200, `{"value":[{"name":"test-model","format":"OpenAI","version":"test-version"}]}`, nil},
	)
	for _, group := range []string{"", "rg-test"} {
		got, err := c.CognitiveServices.Account.List(context.Background(), group)
		if err != nil || len(got) != 1 {
			t.Fatalf("accounts=%v err=%v", got, err)
		}
	}
	one, err := c.CognitiveServices.Account.Show(context.Background(), "rg-test", "ai-test")
	if err != nil || one.Kind == nil || *one.Kind != "AIServices" {
		t.Fatalf("account=%v err=%v", one, err)
	}
	models, err := c.CognitiveServices.Account.ListModels(context.Background(), "rg-test", "ai-test")
	if err != nil || len(models) != 1 {
		t.Fatalf("models=%v err=%v", models, err)
	}
}
func TestRegionalModelsAndQuota(t *testing.T) {
	path := "/subscriptions/" + sub + "/providers/Microsoft.CognitiveServices/locations/eastus"
	c := clientWithSteps(t,
		step{"GET", path + "/models", 200, `{"value":[{"kind":"OpenAI","model":{"name":"test-model","format":"OpenAI","version":"test-version"}}]}`, nil},
		step{"GET", path + "/usages", 200, `{"value":[{"name":{"value":"quota","localizedValue":"Quota"},"currentValue":1,"limit":10,"unit":"Count"}]}`, nil},
	)
	models, err := c.CognitiveServices.Model.List(context.Background(), "eastus")
	if err != nil || len(models) != 1 {
		t.Fatalf("models=%v err=%v", models, err)
	}
	usages, err := c.CognitiveServices.Usage.List(context.Background(), "eastus")
	if err != nil || len(usages) != 1 {
		t.Fatalf("usages=%v err=%v", usages, err)
	}
}
func TestDeploymentRead(t *testing.T) {
	c := clientWithSteps(t,
		step{"GET", accountPath + "/deployments", 200, `{"value":[` + deploymentBody + `]}`, nil},
		step{"GET", deploymentPath, 200, deploymentBody, nil},
		step{"GET", deploymentPath + "/skus", 200, `{"value":[{"sku":{"name":"GlobalStandard","capacity":1}}]}`, nil},
	)
	d := c.CognitiveServices.Account.Deployment
	got, err := d.List(context.Background(), "rg-test", "ai-test")
	if err != nil || len(got) != 1 {
		t.Fatalf("deployments=%v err=%v", got, err)
	}
	one, err := d.Show(context.Background(), "rg-test", "ai-test", "d-test")
	if err != nil || one.Name == nil || *one.Name != "d-test" {
		t.Fatalf("deployment=%v err=%v", one, err)
	}
	skus, err := d.ListSKUs(context.Background(), "rg-test", "ai-test", "d-test")
	if err != nil || len(skus) != 1 {
		t.Fatalf("skus=%v err=%v", skus, err)
	}
}
func createOptions() cs.CreateOptions {
	return cs.CreateOptions{ModelFormat: "OpenAI", ModelName: "test-model", ModelVersion: "test-version", SKUName: "GlobalStandard", SKUCapacity: 1}
}
func TestDeploymentMutations(t *testing.T) {
	c := clientWithSteps(t,
		step{"PUT", deploymentPath, 200, deploymentBody, func(r *http.Request) {
			b := mustJSON(t, r)
			properties := b["properties"].(map[string]any)
			model := properties["model"].(map[string]any)
			sku := b["sku"].(map[string]any)
			if model["format"] != "OpenAI" || model["name"] != "test-model" || model["version"] != "test-version" || sku["capacity"] != float64(1) {
				t.Fatalf("unexpected body: %#v", b)
			}
		}},
		step{"PATCH", deploymentPath, 200, deploymentBody, func(r *http.Request) {
			b := mustJSON(t, r)
			if _, exists := b["properties"]; exists {
				t.Fatal("PATCH unexpectedly overwrites model properties")
			}
			sku := b["sku"].(map[string]any)
			if sku["capacity"] != float64(2) {
				t.Fatalf("unexpected PATCH: %#v", b)
			}
		}},
		step{"DELETE", deploymentPath, 204, "", nil},
	)
	d := c.CognitiveServices.Account.Deployment
	created, err := d.Create(context.Background(), "rg-test", "ai-test", "d-test", createOptions())
	if err != nil || created == nil {
		t.Fatalf("created=%v err=%v", created, err)
	}
	updated, err := d.Update(context.Background(), "rg-test", "ai-test", "d-test", cs.UpdateOptions{SKUName: "GlobalStandard", SKUCapacity: 2})
	if err != nil || updated == nil {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
	if err := d.Delete(context.Background(), "rg-test", "ai-test", "d-test"); err != nil {
		t.Fatal(err)
	}
}
func TestErrorsRemainInspectable(t *testing.T) {
	c := clientWithSteps(t, step{"GET", deploymentPath, 403, `{"error":{"code":"AuthorizationFailed","message":"unit test denial"}}`, nil})
	_, err := c.CognitiveServices.Account.Deployment.Show(context.Background(), "rg-test", "ai-test", "d-test")
	var re *azcore.ResponseError
	if !errors.As(err, &re) || re.StatusCode != 403 || re.ErrorCode != "AuthorizationFailed" {
		t.Fatalf("lost response error: %v", err)
	}
}
func TestListFailureDoesNotReturnPartialData(t *testing.T) {
	next := "https://management.azure.com/subscriptions?api-version=2022-12-01&$skiptoken=next"
	c := clientWithSteps(t,
		step{"GET", "/subscriptions", 200, `{"value":[{"subscriptionId":"` + sub + `"}],"nextLink":"` + next + `"}`, nil},
		step{"GET", "/subscriptions", 403, `{"error":{"code":"AuthorizationFailed","message":"unit test denial"}}`, nil},
	)
	got, err := c.Account.List(context.Background())
	if err == nil || got != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
func TestCredentialFailureIsReturned(t *testing.T) {
	cause := errors.New("please sign in")
	c := newTestClient(t, sub, &fakeCredential{cause: cause}, transportFunc(func(*http.Request) (*http.Response, error) { t.Error("HTTP after auth failure"); return nil, cause }))
	_, err := c.Group.List(context.Background())
	if !errors.Is(err, cause) {
		t.Fatalf("lost cause: %v", err)
	}
}
func TestInvalidInputDoesNotSendRequests(t *testing.T) {
	c := clientWithSteps(t)
	ctx := context.Background()
	d := c.CognitiveServices.Account.Deployment
	if _, err := c.Group.Show(ctx, " "); err == nil {
		t.Fatal("accepted empty group")
	}
	if _, err := d.Create(ctx, "rg-test", "ai-test", "d-test", cs.CreateOptions{}); err == nil {
		t.Fatal("accepted empty create")
	}
	for _, bad := range []int32{0, -1} {
		o := createOptions()
		o.SKUCapacity = bad
		if _, err := d.Create(ctx, "rg-test", "ai-test", "d-test", o); err == nil {
			t.Fatal("accepted invalid capacity")
		}
		if _, err := d.Update(ctx, "rg-test", "ai-test", "d-test", cs.UpdateOptions{SKUName: "GlobalStandard", SKUCapacity: bad}); err == nil {
			t.Fatal("accepted invalid update")
		}
	}
	if err := d.Delete(ctx, "rg-test", "ai-test", ""); err == nil {
		t.Fatal("accepted empty deletion target")
	}
}
func TestCreateWaitsForLongRunningOperation(t *testing.T) {
	var puts, polls, gets atomic.Int32
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == "PUT" && r.URL.Path == deploymentPath {
			puts.Add(1)
			resp := response(r, 201, strings.Replace(deploymentBody, "Succeeded", "Creating", 1))
			resp.Header.Set("Azure-AsyncOperation", "https://management.azure.com/operations/test?api-version=2025-09-01")
			resp.Header.Set("Retry-After", "1")
			return resp, nil
		}
		if r.Method == "GET" && r.URL.Path == "/operations/test" {
			polls.Add(1)
			return response(r, 200, `{"status":"Succeeded"}`), nil
		}
		if r.Method == "GET" && r.URL.Path == deploymentPath {
			gets.Add(1)
			return response(r, 200, deploymentBody), nil
		}
		return nil, fmt.Errorf("unexpected LRO request: %s %s", r.Method, r.URL)
	})
	c := newTestClient(t, sub, &fakeCredential{}, transport)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := c.CognitiveServices.Account.Deployment.Create(ctx, "rg-test", "ai-test", "d-test", createOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got.Name == nil || *got.Name != "d-test" || puts.Load() != 1 || polls.Load() < 1 || gets.Load() < 1 {
		t.Fatalf("LRO not completed: %+v PUT=%d poll=%d GET=%d", got, puts.Load(), polls.Load(), gets.Load())
	}
}
func TestCancellationStopsWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var requests atomic.Int32
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		resp := response(r, 201, strings.Replace(deploymentBody, "Succeeded", "Creating", 1))
		resp.Header.Set("Azure-AsyncOperation", "https://management.azure.com/operations/test?api-version=2025-09-01")
		resp.Header.Set("Retry-After", "1")
		cancel()
		return resp, nil
	})
	c := newTestClient(t, sub, &fakeCredential{}, transport)
	_, err := c.CognitiveServices.Account.Deployment.Create(ctx, "rg-test", "ai-test", "d-test", createOptions())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("unexpected follow-up request after cancel: %d", requests.Load())
	}
}
