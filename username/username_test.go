package username

import (
	"errors"
	"testing"
)

func TestIsValid(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"ab", true},              // min length 2
		{"robertpaulson", true},   // typical
		{"a_b-c", true},           // underscore + hyphen allowed
		{"happy-otter-1234", true}, // auto-assigned shape is still valid format
		{"user123", true},
		{"a", false},                     // too short (1)
		{"", false},                      // empty
		{"AB", false},                    // uppercase
		{"user name", false},             // space
		{"user.name", false},             // dot
		{"user@host", false},             // at-sign
		{"héllo", false},                 // non-ascii
		{string(make([]byte, 51)), false}, // way too long (also null bytes)
	}
	for _, c := range cases {
		if got := IsValid(c.name); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.name, got, c.want)
		}
	}
	// exactly 50 chars is valid, 51 is not
	fifty := ""
	for i := 0; i < 50; i++ {
		fifty += "a"
	}
	if !IsValid(fifty) {
		t.Errorf("IsValid(50 chars) = false, want true")
	}
	if IsValid(fifty + "a") {
		t.Errorf("IsValid(51 chars) = true, want false")
	}
}

func TestIsAutoAssigned(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"happy-otter-1234", true},
		{"golden-falcon-9999", true},
		{"a-b-0000", true},
		{"happy-otter-123", false},   // 3 digits
		{"happy-otter-12345", false}, // 5 digits
		{"happy_otter_1234", false},  // underscores not hyphens
		{"happyotter1234", false},    // no separators
		{"happy-otter", false},       // no number
		{"happy-otter-abcd", false},  // letters not digits
		{"1happy-otter-1234", false}, // leading digit in adjective slot
		{"happy-otter-cub-1234", false}, // extra segment
		{"robertpaulson", false},     // a real human name
	}
	for _, c := range cases {
		if got := IsAutoAssigned(c.name); got != c.want {
			t.Errorf("IsAutoAssigned(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIsValidHumanName(t *testing.T) {
	// Valid human names: valid format AND not the reserved auto-assigned shape.
	if !IsValidHumanName("robertpaulson") {
		t.Error("IsValidHumanName(robertpaulson) = false, want true")
	}
	if !IsValidHumanName("alice") {
		t.Error("IsValidHumanName(alice) = false, want true")
	}
	// Squatting the auto-assign namespace must be blocked even though the format is valid.
	if IsValidHumanName("happy-otter-1234") {
		t.Error("IsValidHumanName(happy-otter-1234) = true, want false (reserved shape)")
	}
	// Invalid format is not a valid human name.
	if IsValidHumanName("a") {
		t.Error("IsValidHumanName(a) = true, want false (too short)")
	}
	if IsValidHumanName("Bad Name") {
		t.Error("IsValidHumanName(Bad Name) = true, want false")
	}
}

func TestGenerate_Shape(t *testing.T) {
	name, err := Generate(func(string) (bool, error) { return true, nil })
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !IsValid(name) {
		t.Errorf("Generate produced %q which fails IsValid", name)
	}
	if !IsAutoAssigned(name) {
		t.Errorf("Generate produced %q which fails IsAutoAssigned", name)
	}
	if IsValidHumanName(name) {
		t.Errorf("Generate produced %q which a human could claim (must not)", name)
	}
}

func TestGenerate_CollisionRetry(t *testing.T) {
	// First two candidates "taken", third free — Generate must skip and succeed.
	calls := 0
	name, err := Generate(func(string) (bool, error) {
		calls++
		return calls > 2, nil
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 availability checks, got %d", calls)
	}
	if !IsAutoAssigned(name) {
		t.Errorf("Generate produced %q which fails IsAutoAssigned", name)
	}
}

func TestGenerate_Exhausted(t *testing.T) {
	// Nothing ever available -> error after maxGenerateAttempts.
	calls := 0
	_, err := Generate(func(string) (bool, error) {
		calls++
		return false, nil
	})
	if err == nil {
		t.Fatal("Generate() expected error when no candidate is available, got nil")
	}
	if calls != maxGenerateAttempts {
		t.Errorf("expected %d attempts, got %d", maxGenerateAttempts, calls)
	}
}

func TestGenerate_AvailError(t *testing.T) {
	sentinel := errors.New("db down")
	_, err := Generate(func(string) (bool, error) {
		return false, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("Generate() error = %v, want wrapped %v", err, sentinel)
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	// Sanity: across many generations we get some variety (not a constant).
	seen := map[string]int{}
	for i := 0; i < 200; i++ {
		n, err := Generate(func(string) (bool, error) { return true, nil })
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		seen[n]++
	}
	if len(seen) < 50 {
		t.Errorf("expected varied output, only %d distinct of 200", len(seen))
	}
}
