package debversion

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/kelleyk/gokk"
)

type DebianVersion struct {
	// Ref.: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
	//
	// notes:
	// - a 0 epoch is the same as no epoch
	// - no debian revision is the same as "-0"

	Epoch           string
	UpstreamVersion string
	DebianRevision  string
}

func FromString(s string) (DebianVersion, error) {
	epoch, _, s := gokk.Partition(s, ":")
	upstream, _, debian := gokk.PartitionLast(s, "-")
	// N.B.: We deliberately do not normalize things here.
	return DebianVersion{
		Epoch:           epoch,
		UpstreamVersion: upstream,
		DebianRevision:  debian,
	}, nil
}

func (v DebianVersion) String() string {
	s := v.UpstreamVersion
	if v.DebianRevision != "" {
		s = fmt.Sprintf("%s-%s", s, v.DebianRevision)
	}
	if v.Epoch != "" {
		s = fmt.Sprintf("%s:%s", v.Epoch, s)
	}
	return s
}

func (v DebianVersion) normEpoch() string {
	if v.Epoch == "" {
		return "0"
	}
	return v.Epoch
}

func (v DebianVersion) normUpstreamVersion() string {
	return v.UpstreamVersion
}

func (v DebianVersion) normDebianRevision() string {
	if v.DebianRevision == "" {
		return "0"
	}
	return v.DebianRevision
}

type Result int

const (
	ResultUnknown Result = iota
	ResultLess
	ResultGreater
	ResultEqual
	// ResultError // implies that an internal error happened
)

func (r Result) String() string {
	switch r {
	case ResultLess:
		return "LESS"
	case ResultGreater:
		return "GREATER"
	case ResultEqual:
		return "EQUAL"
	default:
		return "INVALID"
	}
}

// Ref. 'DoCmpVersion()' in /apt-pkg/deb/debversion.cc from libapt.
func doCmpVersion(a, b DebianVersion) Result {
	// From libapt: "This fragments the version into E:V-R triples and compares each
	// portion separately."
	//
	// @KK: E="epoch", V="upstream version", R="debian version"

	switch r := doCmpFragment(a.normEpoch(), b.normEpoch()); r {
	case ResultEqual:
	case ResultLess, ResultGreater:
		return r
	default:
		panic(fmt.Sprintf("unexpected result: %v", r))
	}

	switch r := doCmpFragment(a.normUpstreamVersion(), b.normUpstreamVersion()); r {
	case ResultEqual:
	case ResultLess, ResultGreater:
		return r
	default:
		panic(fmt.Sprintf("unexpected result: %v", r))
	}

	return doCmpFragment(a.normDebianRevision(), b.normDebianRevision())
}

// Ref. 'DoCmpFragment()' in /apt-pkg/deb/debversion.cc from libapt.
//
// From libapt:
// "Iterate over the whole string
//       What this does is to split the whole string into groups of
//       numeric and non numeric portions. For instance:
//          a67bhgs89
//       Has 4 portions 'a', '67', 'bhgs', '89'. A more normal:
//          2.7.2-linux-1
//       Has '2', '.', '7', '.' ,'-linux-','1'
//  "
//
// static int order(char c)
// {
// 	if (isdigit(c))
//    return 0;
// 	else if (isalpha(c))
//    return c;
// 	else if (c == '~')
// 	  return -1;
// 	else if (c)
// 	  return c + 256;
// 	else
// 	  return 0;
// }
//
// From the policy manual:
//
//   First the initial part of each string consisting entirely of non-digit characters is determined. These two parts
//   (one of which may be empty) are compared lexically. If a difference is found it is returned. The lexical comparison
//   is a comparison of ASCII values modified so that all the letters sort earlier than all the non-letters and so that
//   a tilde sorts before anything, even the end of a part. For example, the following parts are in sorted order from
//   earliest to latest: ~~, ~~a, ~, the empty part, a.[37]
//
//   Then the initial part of the remainder of each string which consists entirely of digit characters is
//   determined. The numerical values of these two parts are compared, and any difference found is returned as the
//   result of the comparison. For these purposes an empty string (which can only occur at the end of one or both
//   version strings being compared) counts as zero.
//
func doCmpFragment(a, b string) Result {
	// fmt.Printf("**** doCmpFragment: %q <> %q\n", a, b)

	for {
		var ap, bp string

		// XXX: review this block
		if a == "" && b == "" {
			// fmt.Printf("  - break (0)\n")
			// return ResultEqual <-- this is what'll eventually be returned
			break
		}

		// Non-digit phase.
		// fmt.Printf("  - entering non-digit phase; before split: a=%q b=%q\n", a, b)
		ap, a = partitionNonDigit(a)
		bp, b = partitionNonDigit(b)
		// fmt.Printf("    - part: ap=%q bp=%q\n", ap, bp)
		// fmt.Printf("    - rem.:  a=%q  b=%q\n", a, b)
		switch r := doCmpNonDigits(ap, bp); r {
		case ResultEqual:
		default:
			return r
		}

		// // XXX: review this block
		if a == "" && b == "" {
			// fmt.Printf("  - break (1)\n")
			// return ResultEqual <-- this is what'll eventually be returned
			break
		}

		// Digit phase.
		// fmt.Printf("  - entering digit phase; before split: a=%q b=%q\n", a, b)
		ap, a = partitionDigit(a)
		bp, b = partitionDigit(b)
		// fmt.Printf("    - part: ap=%q bp=%q\n", ap, bp)
		// fmt.Printf("    - rem.:  a=%q  b=%q\n", a, b)
		switch {
		case ap == "" && bp == "":
			panic("path should be unreachable (0)")
			// 	if a == "" || b == "" {
			// 		panic(fmt.Sprintf("digit bad (B): %q <> %q", a, b))
			// 	}
			// 	// fmt.Printf("  - ret-equal (2)\n")
			// 	return ResultEqual // XXX:
		case ap == "":
			return ResultLess
		case bp == "":
			return ResultGreater
		}
		// fmt.Printf("  - cmpDigit parts %q <> %q\n", ap, bp)
		apInt, err := strconv.Atoi(ap)
		if err != nil {
			panic(err)
		}
		bpInt, err := strconv.Atoi(bp)
		if err != nil {
			panic(err)
		}
		switch {
		case apInt > bpInt:
			return ResultGreater
		case apInt < bpInt:
			return ResultLess
		}
	}

	// XXX: this block should become unnecessary
	// fmt.Printf("  - doCmpFragment end block (by len): %q <> %q\n", a, b)
	switch {
	case len(a) == len(b):
		return ResultEqual
	default:
		panic("path should be unreachable (1)")
		// case len(a) > len(b):
		// 	return ResultGreater // XXX: should this be greater or less?
		// case len(a) < len(b):
		// 	return ResultLess
	}
}

func doCmpNonDigits(ap, bp string) Result {
	// fmt.Printf("  - cmpNonDigits: %q <> %q\n", ap, bp)

	minLen := len(ap)
	if len(bp) < minLen {
		minLen = len(bp)
	}

	for i := 0; i < minLen; i++ {
		d := order(ap[i]) - order(bp[i])
		switch {
		case d > 0:
			return ResultGreater
		case d < 0:
			return ResultLess
		}
	}

	// Any character sorts before end-of-part EXCEPT '~', which sorts behind it (negative order).
	switch {
	case len(ap) > len(bp):
		if ap[len(bp)] == '~' {
			return ResultLess
		}
		return ResultGreater
	case len(ap) < len(bp):
		if bp[len(ap)] == '~' {
			return ResultGreater
		}
		return ResultLess
	default:
		return ResultEqual
	}
}

// [0-9] (like the C standard library function)
func isDigit(c byte) bool {
	x := int(c)
	return x >= 0x30 && x <= 0x39
}

// [A-Za-z] (like the C standard library function)
func isAlpha(c byte) bool {
	x := int(c)
	switch {
	case x >= 0x41 && x <= 0x5A:
		return true
	case x >= 0x61 && x <= 0x7A:
		return true
	default:
		return false
	}
}

func order(c byte) int {
	switch {
	case isDigit(c): // N.B.: in the C version, we would be seeing the first character of the next part
		panic("should not be digits in a non-digit part")
	case isAlpha(c):
		return int(c) // ascii
	case c == '~':
		return -1
	case c == '\x00': // XXX: in C, this is end-of-string
		panic("should not be null characters in version strings")
	default:
		return int(c) + 256 // ascii
	}
}

func (v DebianVersion) Compare(o DebianVersion) Result {
	return doCmpVersion(v, o)
}

func (v DebianVersion) GreaterThan(o DebianVersion) bool {
	return doCmpVersion(v, o) == ResultGreater
}

func (v DebianVersion) LessThan(o DebianVersion) bool {
	return doCmpVersion(v, o) == ResultLess
}

func (v DebianVersion) Equal(o DebianVersion) bool {
	return doCmpVersion(v, o) == ResultEqual
}

// XXX: here we are using Unicode (runes); in the rest of this package, we use ASCII (bytes)
//
// If all of the runes in s are digits, should return (s, "").
func partitionDigit(s string) (string, string) {
	return gokk.TakeWhile(s, unicode.IsDigit)
}

func partitionNonDigit(s string) (string, string) {
	return gokk.TakeWhile(s, func(r rune) bool { return !unicode.IsDigit(r) })
}
