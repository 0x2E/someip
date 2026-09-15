package main

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oschwald/maxminddb-golang"
)

func writeSource(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func toStrings(nets []*net.IPNet) []string {
	out := make([]string, 0, len(nets))
	for _, n := range nets {
		out = append(out, n.String())
	}
	return out
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "adjacent v4 blocks merge",
			in:   "1.0.0.0/24\n1.0.1.0/24\n",
			want: []string{"1.0.0.0/23"},
		},
		{
			name: "duplicate after Masked is removed",
			in:   "1.2.3.4/24\n1.2.3.0/24\n",
			want: []string{"1.2.3.0/24"},
		},
		{
			// regression for go-cidrman: it emitted a fake 2400:3201::/128 between non-adjacent blocks
			name: "non-adjacent v6 blocks do not produce fake /128",
			in:   "2400:3200::/32\n2400:3202::/32\n",
			want: []string{"2400:3200::/32", "2400:3202::/32"},
		},
		{
			name: "overlapping v6 blocks merge",
			in:   "2400:3200::/32\n2400:3200:8000::/33\n",
			want: []string{"2400:3200::/32"},
		},
		{
			name: "comment and blank lines skipped",
			in:   "# comment\n\n1.0.0.0/8\n",
			want: []string{"1.0.0.0/8"},
		},
		{
			name: "v4 and v6 both kept, sorted",
			in:   "2001:db8::/32\n1.0.0.0/8\n",
			want: []string{"1.0.0.0/8", "2001:db8::/32"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets, err := merge([]string{writeSource(t, tt.in)})
			if err != nil {
				t.Fatalf("merge: %v", err)
			}
			got := toStrings(nets)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeErrors(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantSubstr string
	}{
		{
			name:       "invalid line reports file and line",
			in:         "1.0.0.0/8\ngarbage line\n",
			wantSubstr: ":2:",
		},
		{
			name:       "4-in-6 address rejected",
			in:         "::ffff:1.2.3.0/120\n",
			wantSubstr: "4-in-6",
		},
		{
			name:       "empty result",
			in:         "# only a comment\n",
			wantSubstr: "no valid CIDR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeSource(t, tt.in)
			_, err := merge([]string{path})
			if err == nil {
				t.Fatalf("merge() succeeded, want error containing %q", tt.wantSubstr)
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("merge() error = %q, want substring %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestSaveFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cidr.txt")
	nets, err := merge([]string{writeSource(t, "1.0.1.0/24\n2400:3200::/32\n")})
	if err != nil {
		t.Fatal(err)
	}
	if err := saveFile(nets, path); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(lines, ",") != strings.Join(toStrings(nets), ",") {
		t.Errorf("round trip = %v, want %v", lines, toStrings(nets))
	}
}

func TestBuildMMDBLookup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Country.mmdb")
	nets, err := merge([]string{writeSource(t, "1.0.0.0/8\n2400:3200::/32\n")})
	if err != nil {
		t.Fatal(err)
	}
	if err := buildMMDB(nets, path); err != nil {
		t.Fatal(err)
	}

	db, err := maxminddb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		ip      string
		wantCN  bool
		wantHit bool
	}{
		{"1.1.1.1", true, true},
		{"9.9.9.9", false, false},
		{"2400:3200::1", true, true},
		{"2600::1", false, false},
	}
	for _, tt := range tests {
		var record struct {
			Country struct {
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"country"`
		}
		if err := db.Lookup(net.ParseIP(tt.ip), &record); err != nil {
			t.Fatalf("lookup %s: %v", tt.ip, err)
		}
		if gotCN := record.Country.ISOCode == "CN"; gotCN != tt.wantCN {
			t.Errorf("lookup %s: iso_code=%q, want CN=%v", tt.ip, record.Country.ISOCode, tt.wantCN)
		}
	}
}

func TestAtomicWriteFailurePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	boom := errors.New("boom")
	if err := atomicWrite(path, func(*os.File) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("atomicWrite() error = %v, want %v", err, boom)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "old" {
		t.Errorf("existing file changed to %q", b)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}
