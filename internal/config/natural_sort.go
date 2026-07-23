// natural_sort.go orders file names the way their author numbered them.
//
// Plain lexical order puts "10-deploy.md" and "11-scale.md" ahead of
// "2-install.md", which silently reorders a book whose chapters are numbered
// past nine — and neither build nor validate had any way to notice.
package config

import (
	"cmp"
	"strings"
)

// NaturalCompare compares a and b treating runs of digits as numbers, so
// "2-install" sorts before "10-deploy". It returns a negative number when a
// sorts first, zero when the two are equivalent, and a positive number
// otherwise, matching the contract of [cmp.Compare].
func NaturalCompare(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isASCIIDigit(a[i]) && isASCIIDigit(b[j]) {
			ai, aEnd := digitRun(a, i)
			bj, bEnd := digitRun(b, j)
			if n := compareNumericRun(ai, bj); n != 0 {
				return n
			}
			i, j = aEnd, bEnd
			continue
		}
		if a[i] != b[j] {
			return cmp.Compare(a[i], b[j])
		}
		i++
		j++
	}
	// One string is a prefix of the other, or they differ only in the zero
	// padding skipped above; the shorter one sorts first.
	if n := cmp.Compare(len(a)-i, len(b)-j); n != 0 {
		return n
	}
	return strings.Compare(a, b)
}

// NaturalLess reports whether a sorts before b in natural order.
func NaturalLess(a, b string) bool { return NaturalCompare(a, b) < 0 }

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

// digitRun returns the digit run starting at i with leading zeros removed,
// along with the index just past the run.
func digitRun(s string, i int) (string, int) {
	end := i
	for end < len(s) && isASCIIDigit(s[end]) {
		end++
	}
	return strings.TrimLeft(s[i:end], "0"), end
}

// compareNumericRun compares two zero-stripped digit runs by value. Longer
// means larger, and equal lengths compare lexically, which avoids parsing
// integers that may not fit in one.
func compareNumericRun(a, b string) int {
	if n := cmp.Compare(len(a), len(b)); n != 0 {
		return n
	}
	return strings.Compare(a, b)
}
