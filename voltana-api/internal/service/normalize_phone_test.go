package service

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		// Iranian mobile — 0 + 10 digits (11 total)
		{"09121234567", "+989121234567", false},
		{"09715228900", "+989715228900", false},
		// Iranian with +98 or bare 98
		{"+989121234567", "+989121234567", false},
		{"989121234567", "+989121234567", false},
		// Spaces / dashes stripped
		{"0912 123 4567", "+989121234567", false},
		{"0912-123-4567", "+989121234567", false},

		// UAE — +971 (3-digit country code)
		{"+971522890098", "+971522890098", false},
		// UAE entered as bare digits (country code, no symbol)
		{"971522890098", "+971522890098", false},
		// UAE with 00 international exit code
		{"00971522890098", "+971522890098", false},
		// UAE with leading 0 (non-11-digit Iranian heuristic strips the 0)
		{"0971522890098", "+971522890098", false},

		// US — +1 (1-digit country code)
		{"+12025551234", "+12025551234", false},
		{"0012025551234", "+12025551234", false},

		// Too short → error
		{"12345", "", true},
		// Completely invalid
		{"abc", "", true},
	}

	for _, tc := range cases {
		got, err := normalizePhone(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("normalizePhone(%q) = %q, want error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizePhone(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizePhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
