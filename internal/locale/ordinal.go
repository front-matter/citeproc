package locale

import (
	"fmt"
	"strconv"

	"github.com/front-matter/citeproc/internal/csl"
)

// Ordinal returns the ordinal suffix for n ("nd" for 2 in English), choosing
// the variant for the gender of the target noun with fallback to the neuter
// variant. It implements CSL 1.0.1+ ordinal matching, and the CSL 1.0 scheme
// when the locale defines "ordinal-01".."ordinal-04" but not "ordinal".
func (c *Chain) Ordinal(n int, gender csl.Gender) string {
	if n < 0 {
		n = -n
	}
	legacy := c.legacyOrdinals()
	var best *csl.Term
	bestScore := -1
	for _, t := range c.ordinals {
		if t.GenderForm != "" && t.GenderForm != gender {
			continue
		}
		rank := ordinalRank(t, n, legacy)
		if rank < 0 {
			continue
		}
		// The group decides first ("ordinal-10".."ordinal-99" over
		// "ordinal-00".."ordinal-09" over "ordinal"); within a group the
		// exact gender variant beats the neuter one.
		score := rank * 2
		if gender != "" && t.GenderForm == gender {
			score++
		}
		if score > bestScore {
			best, bestScore = t, score
		}
	}
	if best == nil {
		return ""
	}
	return best.Single
}

// LongOrdinal returns the long ordinal for n ("second" for 2 in English). The
// terms "long-ordinal-01".."long-ordinal-10" cover 1 to 10; other numbers,
// and numbers whose term is undefined, fall back to n plus Ordinal(n).
func (c *Chain) LongOrdinal(n int, gender csl.Gender) string {
	if n >= 1 && n <= 10 {
		name := fmt.Sprintf("long-ordinal-%02d", n)
		for _, lv := range c.levels {
			var neuter, gendered *csl.Term
			for _, t := range lv.loc.Terms {
				if t.Name != name {
					continue
				}
				switch {
				case t.GenderForm == "" && neuter == nil:
					neuter = t
				case gender != "" && t.GenderForm == gender && gendered == nil:
					gendered = t
				}
			}
			if gendered != nil {
				return gendered.Single
			}
			if neuter != nil {
				return neuter.Single
			}
		}
	}
	return strconv.Itoa(n) + c.Ordinal(n, gender)
}

func (c *Chain) legacyOrdinals() bool {
	defined := map[string]bool{}
	for _, t := range c.ordinals {
		defined[t.Name] = true
	}
	return !defined["ordinal"] && defined["ordinal-01"] && defined["ordinal-02"] &&
		defined["ordinal-03"] && defined["ordinal-04"]
}

// ordinalRank returns -1 if t does not apply to n, 0 for "ordinal", 1 for a
// match in "ordinal-00".."ordinal-09" and 2 for "ordinal-10".."ordinal-99".
func ordinalRank(t *csl.Term, n int, legacy bool) int {
	if t.Name == "ordinal" {
		return 0
	}
	num, err := strconv.Atoi(t.Name[len("ordinal-"):])
	if err != nil {
		return -1
	}
	if legacy {
		// CSL 1.0: -01, -02 and -03 for numbers ending in 1, 2 and 3 except
		// 11, 12 and 13; -04 for everything else.
		class := 4
		if d := n % 10; n%100/10 != 1 && d >= 1 && d <= 3 {
			class = d
		}
		if num == class {
			return 1
		}
		return -1
	}
	var ok bool
	if num < 10 {
		switch t.Match {
		case csl.MatchLastTwoDigits:
			ok = n%100 == num
		case csl.MatchWholeNumber:
			ok = n == num
		default:
			ok = n%10 == num
		}
		if ok {
			return 1
		}
		return -1
	}
	if t.Match == csl.MatchWholeNumber {
		ok = n == num
	} else {
		ok = n%100 == num
	}
	if ok {
		return 2
	}
	return -1
}
