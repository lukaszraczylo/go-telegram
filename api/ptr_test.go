package api_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
)

func TestPtr(t *testing.T) {
	if got := api.Ptr[int64](5); got == nil || *got != 5 {
		t.Fatalf("Ptr[int64](5) = %v, want *5", got)
	}
	if got := api.Ptr(false); got == nil || *got != false {
		t.Fatalf("Ptr(false) = %v, want *false", got)
	}
	if got := api.Ptr("hello"); got == nil || *got != "hello" {
		t.Fatalf("Ptr(\"hello\") = %v, want *\"hello\"", got)
	}

	n := int64(42)
	got := api.Ptr(n)
	if got == nil || *got != 42 {
		t.Fatalf("Ptr(n) = %v, want *42", got)
	}
	if got == &n {
		t.Fatalf("Ptr should copy, not alias caller's variable")
	}
}
