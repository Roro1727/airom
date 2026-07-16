package cli

import "testing"

func TestParseSize(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1048576", 1 << 20, false},
		{"512k", 512 << 10, false},
		{"512K", 512 << 10, false},
		{"256m", 256 << 20, false},
		{"256M", 256 << 20, false},
		{"256mb", 256 << 20, false},
		{"2g", 2 << 30, false},
		{"2G", 2 << 30, false},
		{" 64m ", 64 << 20, false},
		{"0", 0, false},
		{"", 0, true},
		{"m", 0, true},
		{"-1m", 0, true},
		{"1.5m", 0, true},
		{"1t", 0, true},
		{"lots", 0, true},
		{"9999999999g", 0, true}, // overflow
	}
	for _, tc := range cases {
		got, err := parseSize(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q): want error, got %d", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseSize(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestFormatSizeRoundTrip(t *testing.T) {
	for _, s := range []string{"256m", "1m", "512k", "2g", "3"} {
		n, err := parseSize(s)
		if err != nil {
			t.Fatalf("parseSize(%q): %v", s, err)
		}
		if got := formatSize(n); got != s {
			t.Errorf("formatSize(parseSize(%q)) = %q", s, got)
		}
	}
}
