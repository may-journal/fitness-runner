package architecture

import (
	"regexp"
)

// cspell:ignore fileprivate typealias
var (
	// swiftDeclaration matches a top-level type, protocol, or function
	// declaration at column zero, after its modifiers and attributes.
	swiftDeclaration = regexp.MustCompile(`^(?:(?:public|open|internal|fileprivate|private|final|indirect|@[A-Za-z_]\w*(?:\([^)]*\))?)\s+)*(?:class|struct|enum|protocol|actor|typealias|func)\s+([A-Za-z_]\w*)`)
	identifier       = regexp.MustCompile(`[A-Za-z_]\w*`)
)

// swiftIndex maps each top-level Swift symbol to the file declaring it.
// A symbol two files declare resolves to neither.
type swiftIndex struct {
	m       *Model
	symbols map[string]string
}

// swiftReferences reads the top-level declarations of every product Swift
// file, then each line that names a symbol another file declares, outside
// comments and string literals. Files in one module share no import, so
// the symbol is the reference.
func (m *Model) swiftReferences() []reference {
	s := &swiftIndex{m: m, symbols: map[string]string{}}
	files := m.productSources(".swift")
	for _, file := range files {
		s.declare(file)
	}
	var refs []reference
	for _, file := range files {
		refs = append(refs, s.references(file)...)
	}
	return refs
}

func (s *swiftIndex) declare(file string) {
	for _, line := range s.m.repo.read(file) {
		if match := swiftDeclaration.FindStringSubmatch(line); match != nil {
			s.symbol(match[1], file)
		}
	}
}

func (s *swiftIndex) symbol(name, file string) {
	if first, seen := s.symbols[name]; seen && first != file {
		s.symbols[name] = ""
		return
	}
	s.symbols[name] = file
}

func (s *swiftIndex) references(file string) []reference {
	var refs []reference
	sc, cond := &scanner{literals: true}, &conditions{}
	for i, line := range s.m.repo.read(file) {
		code := sc.line(line)
		if cond.directive(code) {
			continue
		}
		refs = s.line(file, i+1, code, cond.String(), refs)
	}
	return refs
}

// line adds one reference per symbol the code names, ignoring member
// access after a dot.
func (s *swiftIndex) line(file string, n int, code, condition string, refs []reference) []reference {
	seen := map[string]bool{}
	for _, span := range identifier.FindAllStringIndex(code, -1) {
		name := code[span[0]:span[1]]
		if to := s.symbols[name]; s.names(to, file, name, seen) && !afterDot(code, span[0]) {
			seen[name] = true
			refs = append(refs, reference{At: Location{file, n}, To: to, Symbol: name, Condition: condition})
		}
	}
	return refs
}

// names reports a symbol another file declares, the first time this line
// names it.
func (s *swiftIndex) names(to, file, name string, seen map[string]bool) bool {
	return to != "" && to != file && !seen[name]
}

func afterDot(code string, start int) bool {
	return start > 0 && code[start-1] == '.'
}
