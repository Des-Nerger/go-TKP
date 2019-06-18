package main
import (
	//"fmt"
	//"os"
	//"reflect"
	"sort"
	"strings"
	"unicode"
	//"unicode/utf8"
)

func fieldsN(s string, n int) []string {
	wasSpace:=true
	count:=0
/*
	if r, _ := utf8.DecodeRuneInString(s); !unicode.IsSpace(r) {
		s = "^" + s
	}
	s+=" $"
*/
	return fieldsFunc(s, func(r rune) bool {
		if count<n {
			if !unicode.IsSpace(r) {
				wasSpace = false
				return false
			}
			if !wasSpace {
				count++
				wasSpace = false
			}
			return true
		}
		return false
	})
}

// FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c)
// and returns an array of slices of s. If all code points in s satisfy f(c) or the
// string is empty, an empty slice is returned.
// FieldsFunc makes no guarantees about the order in which it calls f(c).
// If f does not return consistent results for a given c, FieldsFunc may crash.
func fieldsFunc(s string, f func(rune) bool) []string {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	wasField := false
	fromIndex := 0
	for i, rune := range s {
		if f(rune) {
			if wasField {
				spans = append(spans, span{start: fromIndex, end: i})
				wasField = false
			}
		} else {
			if !wasField {
				fromIndex = i
				wasField = true
			}
		}
	}

	// Last field might end at EOF.
	if wasField {
		spans = append(spans, span{fromIndex, len(s)})
	}

	if len(spans)==0 {
		spans = append(spans, span{0, 0})
	}
	switch spansLen:=len(spans); spansLen {
	case 1:
		emptySpan := span{spans[0].end, spans[0].end}
		for i:=spansLen; i<3; i++ {
			spans = append(spans, emptySpan)
		}
	case 2:
		spans = append(spans, spans[1])
	default:
		if s[spans[1].start:spans[1].end]!="=" {
			spans[2].start = spans[1].start
		}
	}

	// Create strings from recorded field indices.
	a := make([]string, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end]
		if i==0 {
			a[i] = strings.Replace(a[i], "^", "^^", -1)
			if span.start==0 {
				a[i] = "^"+a[i]
			}
		}
	}

	return a
}

/*
func capBulder(sb strings.Builder) int {
	return reflect.ValueOf(sb).FieldByName("buf").Cap()
}
*/

type stringSet []string

func makeStringSet(strings []string) stringSet {
	sort.Strings(strings)
	return strings
}
