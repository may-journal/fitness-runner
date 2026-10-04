// Package gocoverage measures entry coverage and reports test overlap.
package gocoverage

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type Block struct {
	File                                       string
	StartLine, StartColumn, EndLine, EndColumn int
	Statements                                 int64
}
type Profile map[Block]bool

func ReadProfile(r io.Reader) (Profile, error) {
	s := bufio.NewScanner(r)
	if !s.Scan() {
		return nil, fmt.Errorf("missing coverage header")
	}
	if !validMode(s.Text()) {
		return nil, fmt.Errorf("invalid coverage mode: %s", s.Text())
	}
	return scanProfile(s)
}
func scanProfile(s *bufio.Scanner) (Profile, error) {
	p := Profile{}
	positions := map[Block]int64{}
	for s.Scan() {
		b, covered, err := readBlock(s.Text())
		if err != nil {
			return nil, err
		}
		if err := checkPosition(positions, b); err != nil {
			return nil, err
		}
		p[b] = p[b] || covered
	}
	return p, s.Err()
}
func validMode(s string) bool {
	return s == "mode: set" || s == "mode: count" || s == "mode: atomic"
}
func readBlock(s string) (Block, bool, error) {
	fields := strings.Fields(s)
	if len(fields) != 3 {
		return Block{}, false, fmt.Errorf("invalid coverage record: %q", s)
	}
	pos := strings.LastIndex(fields[0], ":")
	if pos <= 0 {
		return Block{}, false, fmt.Errorf("missing coverage position: %q", s)
	}
	b, err := blockPosition(fields[0][:pos], fields[0][pos+1:])
	if err != nil {
		return b, false, err
	}
	return blockCounts(b, fields[1], fields[2])
}

var positionPattern = regexp.MustCompile(`^(\d+)\.(\d+),(\d+)\.(\d+)$`)

func blockPosition(file, position string) (Block, error) {
	b := Block{File: file}
	parts := positionPattern.FindStringSubmatch(position)
	if len(parts) != 5 {
		return b, fmt.Errorf("invalid coverage range: %s", position)
	}
	values := []*int{&b.StartLine, &b.StartColumn, &b.EndLine, &b.EndColumn}
	for i, value := range parts[1:] {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return b, fmt.Errorf("invalid coverage coordinate: %s", value)
		}
		*values[i] = n
	}
	return b, validRange(b)
}
func validRange(b Block) error {
	if b.EndLine < b.StartLine || (b.EndLine == b.StartLine && b.EndColumn < b.StartColumn) {
		return fmt.Errorf("reversed coverage range in %s", b.File)
	}
	return nil
}
func blockCounts(b Block, statements, count string) (Block, bool, error) {
	n, err := strconv.ParseInt(statements, 10, 64)
	if err != nil || n < 0 {
		return b, false, fmt.Errorf("invalid statement count: %s", statements)
	}
	hits, err := strconv.ParseInt(count, 10, 64)
	if err != nil || hits < 0 {
		return b, false, fmt.Errorf("invalid hit count: %s", count)
	}
	b.Statements = n
	return b, hits > 0, nil
}
func (p Profile) Merge(other Profile) error {
	positions := make(map[Block]int64, len(p))
	for b := range p {
		positions[positionKey(b)] = b.Statements
	}
	for b, hit := range other {
		if err := checkPosition(positions, b); err != nil {
			return err
		}
		p[b] = p[b] || hit
	}
	return nil
}
func positionKey(b Block) Block { b.Statements = 0; return b }
func checkPosition(positions map[Block]int64, b Block) error {
	key := positionKey(b)
	if n, ok := positions[key]; ok && n != b.Statements {
		return fmt.Errorf("conflicting coverage counts in %s", b.File)
	}
	positions[key] = b.Statements
	return nil
}
