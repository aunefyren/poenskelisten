package oauth

import (
	"reflect"
	"testing"
)

func TestIsValidAndLookup(t *testing.T) {
	if !IsValid("openid") {
		t.Error("openid should be valid")
	}
	if IsValid("mcp:does.not.exist") {
		t.Error("unknown scope should be invalid")
	}
	if s, ok := Lookup("mcp:wishlists.read"); !ok || !s.MCP {
		t.Error("mcp:wishlists.read should be a known MCP scope")
	}
}

func TestMCPNames(t *testing.T) {
	for _, name := range MCPNames() {
		if s, ok := Lookup(name); !ok || !s.MCP {
			t.Errorf("MCPNames returned non-MCP scope %q", name)
		}
	}
	if len(MCPNames()) == 0 {
		t.Error("expected at least one MCP scope")
	}
}

func TestParseAndFilterValid(t *testing.T) {
	got := FilterValid(Parse("openid  email openid bogus mcp:groups.read"))
	want := []string{"openid", "email", "mcp:groups.read"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterValid = %v, want %v", got, want)
	}
}

func TestAllNamesCoversRegistry(t *testing.T) {
	if len(AllNames()) != len(Registry) {
		t.Errorf("AllNames returned %d names, want %d", len(AllNames()), len(Registry))
	}
}
