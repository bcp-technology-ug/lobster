package builtin

import (
	"fmt"
	"strings"
)

// splitArgs splits a shell-like argument string into a slice of tokens.
// It handles:
//   - unquoted tokens separated by whitespace
//   - single-quoted literals: 'hello world' → one token, no escape processing
//   - double-quoted strings: "hello \"world\"" → escape sequences \" and \\ honoured
//
// This is intentionally minimal — it covers the lobster CLI flag syntax
// (e.g. --tags "@smoke", --env "BASE_URL=http://localhost:8080") without
// implementing a full POSIX shell grammar.
func splitArgs(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(ch)
			}

		case inDouble:
			if ch == '"' {
				inDouble = false
			} else if ch == '\\' && i+1 < len(s) {
				next := s[i+1]
				if next == '"' || next == '\\' {
					cur.WriteByte(next)
					i++
				} else {
					cur.WriteByte(ch)
				}
			} else {
				cur.WriteByte(ch)
			}

		case ch == '\'':
			inSingle = true

		case ch == '"':
			inDouble = true

		case ch == ' ' || ch == '\t':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}

		default:
			cur.WriteByte(ch)
		}
	}

	if inSingle {
		return nil, fmt.Errorf("splitArgs: unterminated single quote in: %s", s)
	}
	if inDouble {
		return nil, fmt.Errorf("splitArgs: unterminated double quote in: %s", s)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
