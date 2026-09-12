package check

import (
	"errors"
	"testing"
)

func TestSubscriptionID(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"00000000-0000-0000-0000-000000000001", true},
		{"ABCDEF01-0000-0000-0000-000000000001", true},
		{"", false}, {"my-subscription", false}, {"../other", false},
		{"00000000-0000-0000-0000-00000000000Z", false},
		{" 00000000-0000-0000-0000-000000000001", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			if got := SubscriptionID(tc.value); (got == nil) != tc.valid {
				t.Fatalf("got %v, valid=%v", got, tc.valid)
			}
		})
	}
	if !errors.Is(SubscriptionID(""), ErrSubscriptionRequired) {
		t.Fatal("missing sentinel")
	}
}

func TestRequired(t *testing.T) {
	for _, s := range []string{"", " ", "\n", " rg", "rg "} {
		if Required("name", s) == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if err := Required("name", "rg-test"); err != nil {
		t.Fatal(err)
	}
}

func TestScopeAndNames(t *testing.T) {
	if err := Scope(true); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(Scope(false), ErrSubscriptionRequired) {
		t.Fatal("missing sentinel")
	}
	if Resource("rg", "a") != nil || Deployment("rg", "a", "d") != nil {
		t.Fatal("valid names rejected")
	}
	for _, args := range [][3]string{{"", "a", "d"}, {"rg", "", "d"}, {"rg", "a", ""}} {
		if Deployment(args[0], args[1], args[2]) == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
