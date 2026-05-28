package migrations

import "testing"

func TestLoadEmbeddedMigrations(t *testing.T) {
	ms, err := load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(ms) == 0 {
		t.Fatal("expected at least one embedded migration")
	}
	for i := 1; i < len(ms); i++ {
		if ms[i].version <= ms[i-1].version {
			t.Fatalf("migrations not strictly ascending: %d then %d", ms[i-1].version, ms[i].version)
		}
	}
	if ms[0].version != 1 {
		t.Fatalf("first migration must be version 1, got %d", ms[0].version)
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		in        string
		wantVer   int
		wantName  string
		wantError bool
	}{
		{"0001_initial.sql", 1, "initial", false},
		{"0042_add_index.sql", 42, "add_index", false},
		{"bad.sql", 0, "", true},
		{"0000_zero.sql", 0, "", true},
		{"xyz_name.sql", 0, "", true},
	}
	for _, tc := range tests {
		m, err := parseName(tc.in)
		if tc.wantError {
			if err == nil {
				t.Errorf("%s: expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.in, err)
			continue
		}
		if m.version != tc.wantVer || m.name != tc.wantName {
			t.Errorf("%s: got (%d, %q), want (%d, %q)", tc.in, m.version, m.name, tc.wantVer, tc.wantName)
		}
	}
}
