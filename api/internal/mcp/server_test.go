package mcp

import "testing"

func TestCanonicalToolName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "list_subjects", want: "list_subjects"},
		{input: "brights_list_subjects", want: "list_subjects"},
		{input: "list_words", want: "search_words"},
		{input: "brights_search_words", want: "search_words"},
		{input: "list_plans", want: "list_membership_plans"},
		{input: "brights_list_membership_plans", want: "list_membership_plans"},
	}

	for _, tc := range tests {
		if got := canonicalToolName(tc.input); got != tc.want {
			t.Fatalf("canonicalToolName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNewToolErrorResult(t *testing.T) {
	result := newToolErrorResult("brights_search_words", assertErr("membership required"))

	if !result.IsError {
		t.Fatalf("expected IsError to be true")
	}
	if result.StructuredContent["success"] != false {
		t.Fatalf("expected success=false, got %#v", result.StructuredContent["success"])
	}
	if result.StructuredContent["tool"] != "search_words" {
		t.Fatalf("expected canonical tool name, got %#v", result.StructuredContent["tool"])
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
