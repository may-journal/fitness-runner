package architecture

import (
	"fmt"
	"strings"
)

// Reasons a Swift name the module may declare stays unresolved, in the
// order the note lists them.
const (
	unresolvedType      = "type unknown"
	unresolvedImplicit  = "implicit member"
	unresolvedAmbiguous = "ambiguous overload"
)

var unresolvedOrder = []string{unresolvedType, unresolvedImplicit, unresolvedAmbiguous}

// unresolved counts, by reason, the Swift names that may reach another
// file but that the parser could not pin to one declaration. They add no
// edge and fail nothing; the count only shows what the check cannot see.
type unresolved map[string]int

func (u unresolved) add(reason string) {
	if reason != "" {
		u[reason]++
	}
}

// note renders the count, such as "3 Swift references unresolved (2 type
// unknown, 1 implicit member)"; "" when every name resolved.
func (u unresolved) note() string {
	total, parts := 0, []string{}
	for _, reason := range unresolvedOrder {
		if n := u[reason]; n > 0 {
			total += n
			parts = append(parts, fmt.Sprintf("%d %s", n, reason))
		}
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d Swift %s unresolved (%s)", total, plural(total, "reference"), strings.Join(parts, ", "))
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// unresolvedKind says why a use that found nothing stays unresolved: its
// candidates sat in two files, or it is a member another file declares
// whose base type, or context for an implicit member, the parser cannot
// infer. A name no other file declares is external, not unresolved.
func (m *swiftModule) unresolvedKind(u *swiftUse, found []*swiftSymbol) string {
	switch {
	case len(found) > 0:
		return ""
	case m.ambiguous[u]:
		return unresolvedAmbiguous
	case !m.memberElsewhere(u):
		return ""
	case u.implicit:
		return unresolvedImplicit
	}
	return m.unknownBase(u)
}

func (m *swiftModule) unknownBase(u *swiftUse) string {
	if u.member && m.typeOf(u.base, 1) == "" {
		return unresolvedType
	}
	return ""
}

// memberElsewhere reports a member whose name a type in another file
// declares, or an implicit member naming a static member or enum case
// there, the only members an implicit one can be.
func (m *swiftModule) memberElsewhere(u *swiftUse) bool {
	files := map[bool][]string{true: m.staticFiles[u.name], false: m.memberFiles[u.name]}[u.implicit]
	if !u.member && !u.implicit {
		return false
	}
	for _, file := range files {
		if file != u.file {
			return true
		}
	}
	return false
}
