package version

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"testing"
)

// TestParseSemantic ensures version parsing never panics and yields a
// fixed-length [major, minor, patch] slice for anything the Go module system
// can hand us — including the pseudo-version that previously caused an
// index-out-of-range panic.
func TestParseSemantic(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    []int
	}{
		{"tagged", "v0.3.0", []int{0, 3, 0}},
		{"tagged no v", "1.2.3", []int{1, 2, 3}},
		{"pseudo-version", "v0.3.1-0.20260718181500-abcdef123456", []int{0, 3, 1}},
		{"pre-release", "v1.4.0-rc1", []int{1, 4, 0}},
		{"incompatible", "v2.0.0+incompatible", []int{2, 0, 0}},
		{"devel", "(devel)", []int{0, 0, 0}},
		{"empty", "", []int{0, 0, 0}},
		{"too many fields", "v1.2.3.4.5", []int{1, 2, 3}},
		{"short", "v0.5", []int{0, 5, 0}},
		{"garbage", "not-a-version", []int{0, 0, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSemantic(tc.version)

			if len(got) != semVerLen {
				t.Fatalf("parseSemantic(%q) length = %d, want %d", tc.version, len(got), semVerLen)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseSemantic(%q) = %v, want %v", tc.version, got, tc.want)
				}
			}
		})
	}
}

// TestSemanticDoesNotPanic guards the exported entry point: whatever version the
// test binary reports, Semantic must return a well-formed slice and never panic.
func TestSemanticDoesNotPanic(t *testing.T) {
	got := Semantic()
	if len(got) != semVerLen {
		t.Fatalf("Semantic() length = %d, want %d", len(got), semVerLen)
	}
}
