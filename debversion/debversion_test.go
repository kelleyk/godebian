package debversion

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmpFragment(t *testing.T) {
	for _, tt := range []struct {
		a        string
		expected Result
		b        string
	}{
		{"", ResultEqual, ""},
		{"7.6p2", ResultGreater, "7.6"},
	} {
		result := doCmpFragment(tt.a, tt.b)
		assert.Equal(t, tt.expected, result,
			fmt.Sprintf("comparing %q to %q -- expected: %s; actual: %s", tt.a, tt.b, tt.expected, result))
	}
}

func TestCmpNonDigits(t *testing.T) {
	for _, tt := range []struct {
		a        string
		expected Result
		b        string
	}{
		{"", ResultEqual, ""},
		{"pre", ResultLess, "pree"},
		{"foo", ResultGreater, "foo~"},
	} {
		result := doCmpNonDigits(tt.a, tt.b)
		assert.Equal(t, tt.expected, result,
			fmt.Sprintf("comparing %q to %q -- expected: %s; actual: %s", tt.a, tt.b, tt.expected, result))
	}
}

// Cases are borrowed from /test/libapt/compareversion_test.cc in libapt.
func TestCompareVersion(t *testing.T) {
	for _, tt := range []struct {
		a        string
		expected Result
		b        string
	}{
		// Basic tests
		{"7.6p2-4", ResultGreater, "7.6-0"},
		{"1.0.3-3", ResultGreater, "1.0-1"},
		{"1.3", ResultGreater, "1.2.2-2"},
		{"1.3", ResultGreater, "1.2.2"},
		{"0-pre", ResultEqual, "0-pre"},
		{"0-pre", ResultLess, "0-pree"},

		{"1.1.6r2-2", ResultGreater, "1.1.6r-1"},
		{"2.6b2-1", ResultGreater, "2.6b-2"},

		{"98.1p5-1", ResultLess, "98.1-pre2-b6-2"},
		{"0.4a6-2", ResultGreater, "0.4-1"},

		{"1:3.0.5-2", ResultLess, "1:3.0.5.1"},

		// TEST(CompareVersionTest,Epochs)
		{"1:0.4", ResultGreater, "10.3"},
		{"1:1.25-4", ResultLess, "1:1.25-8"},
		{"0:1.18.36", ResultEqual, "1.18.36"},

		{"1.18.36", ResultGreater, "1.18.35"},
		{"0:1.18.36", ResultGreater, "1.18.35"},

		// TEST(CompareVersionTest,Strangeness)
		// Funky, but allowed, characters in upstream version
		{"9:1.18.36:5.4-20", ResultLess, "10:0.5.1-22"},
		{"9:1.18.36:5.4-20", ResultLess, "9:1.18.36:5.5-1"},
		{"9:1.18.36:5.4-20", ResultLess, " 9:1.18.37:4.3-22"},
		{"1.18.36-0.17.35-18", ResultGreater, "1.18.36-19"},
		// Junk
		{"1:1.2.13-3", ResultLess, "1:1.2.13-3.1"},
		{"2.0.7pre1-4", ResultLess, "2.0.7r-1"},
		// if a version includes a dash, it should be the debrev dash - policy says so…
		{"0:0-0-0", ResultGreater, "0-0"},
		// do we like strange versions? Yes we like strange versions…
		{"0", ResultEqual, "0"},
		{"0", ResultEqual, "00"},

		// TEST(CompareVersionTest,DebianBug)
		// #205960
		{"3.0~rc1-1", ResultLess, "3.0-1"},
		// #573592 - debian policy 5.6.12
		{"1.0", ResultEqual, "1.0-0"},
		{"0.2", ResultLess, "1.0-0"},
		{"1.0", ResultLess, "1.0-0+b1"},
		{"1.0", ResultGreater, "1.0-0~"},

		// TEST(CompareVersionTest,CuptTests)
		// "steal" the testcases from (old perl) cupt
		{"1.2.3", ResultEqual, "1.2.3"},                           // identical
		{"4.4.3-2", ResultEqual, "4.4.3-2"},                       // identical
		{"1:2ab:5", ResultEqual, "1:2ab:5"},                       // this is correct...
		{"7:1-a:b-5", ResultEqual, "7:1-a:b-5"},                   // and this
		{"57:1.2.3abYZ+~-4-5", ResultEqual, "57:1.2.3abYZ+~-4-5"}, // and those too
		{"1.2.3", ResultEqual, "0:1.2.3"},                         // zero epoch
		{"1.2.3", ResultEqual, "1.2.3-0"},                         // zero revision
		{"009", ResultEqual, "9"},                                 // zeroes…
		{"009ab5", ResultEqual, "9ab5"},                           // there as well
		{"1.2.3", ResultLess, "1.2.3-1"},                          // added non-zero revision
		{"1.2.3", ResultLess, "1.2.4"},                            // just bigger
		{"1.2.4", ResultGreater, "1.2.3"},                         // order doesn't matter
		{"1.2.24", ResultGreater, "1.2.3"},                        // bigger, eh?
		{"0.10.0", ResultGreater, "0.8.7"},                        // bigger, eh?
		{"3.2", ResultGreater, "2.3"},                             // major number rocks
		{"1.3.2a", ResultGreater, "1.3.2"},                        // letters rock
		{"0.5.0~git", ResultLess, "0.5.0~git2"},                   // numbers rock
		{"2a", ResultLess, "21"},                                  // but not in all places
		{"1.3.2a", ResultLess, "1.3.2b"},                          // but there is another letter
		{"1:1.2.3", ResultGreater, "1.2.4"},                       // epoch rocks
		{"1:1.2.3", ResultLess, "1:1.2.4"},                        // bigger anyway
		{"1.2a+~bCd3", ResultLess, "1.2a++"},                      // tilde doesn't rock
		{"1.2a+~bCd3", ResultGreater, "1.2a+~"},                   // but first is longer!
		{"5:2", ResultGreater, "304-2"},                           // epoch rocks
		{"5:2", ResultLess, "304:2"},                              // so big epoch?
		{"25:2", ResultGreater, "3:2"},                            // 25 > 3, obviously
		{"1:2:123", ResultLess, "1:12:3"},                         // 12 > 2
		{"1.2-5", ResultLess, "1.2-3-5"},                          // 1.2 < 1.2-3
		{"5.10.0", ResultGreater, "5.005"},                        // preceding zeroes don't matters
		{"3a9.8", ResultLess, "3.10.2"},                           // letters are before all letter symbols
		{"3a9.8", ResultGreater, "3~10"},                          // but after the tilde
		{"1.4+OOo3.0.0~", ResultLess, "1.4+OOo3.0.0-4"},           // another tilde check
		{"2.4.7-1", ResultLess, "2.4.7-z"},                        // revision comparing
		{"1.002-1+b2", ResultGreater, "1.00"},                     // whatever...
		// disabled as dpkg doesn't like them… (versions with illegal char)
		// {"2.2.4-47978_Debian_lenny", ResultEqual, "2.2.4-47978_Debian_lenny"}, // and underscore...
	} {
		// log.Printf("** test case: %q <> %q", tt.a, tt.b)
		av, aerr := FromString(tt.a)
		bv, berr := FromString(tt.b)
		if assert.Nil(t, aerr) && assert.Nil(t, berr) {
			result := av.Compare(bv)
			assert.Equal(t, tt.expected, result,
				fmt.Sprintf("comparing %q to %q -- expected: %s; actual: %s", tt.a, tt.b, tt.expected, result))
		}

		avs, bvs := av.String(), bv.String()
		assert.Equal(t, tt.a, avs, fmt.Sprintf("String() failed -- expected: %q, actual: %q", tt.a, avs))
		assert.Equal(t, tt.b, bvs, fmt.Sprintf("String() failed -- expected: %q, actual: %q", tt.b, bvs))
	}
}

func TestPartitionDigit(t *testing.T) {
	p, r := partitionDigit("6")
	assert.Equal(t, "6", p)
	assert.Equal(t, "", r)
}
