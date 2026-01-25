package swagexport

import (
	"fmt"
	"regexp"
	"strings"
)

type excluder struct {
	patterns []string
	res      []*regexp.Regexp
}

func newExcluder(patterns []string) (*excluder, error) {
	var cleaned []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cleaned = append(cleaned, p)
	}
	if len(cleaned) == 0 {
		return nil, nil
	}

	var res []*regexp.Regexp
	for _, p := range cleaned {
		re, err := globToRegexp(p)
		if err != nil {
			return nil, fmt.Errorf("无效 exclude-glob: %s: %w", p, err)
		}
		res = append(res, re)
	}

	return &excluder{
		patterns: cleaned,
		res:      res,
	}, nil
}

func (e *excluder) IsExcluded(path string) bool {
	if e == nil {
		return false
	}
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	for _, re := range e.res {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

func globToRegexp(glob string) (*regexp.Regexp, error) {
	glob = strings.TrimSpace(glob)
	glob = strings.TrimPrefix(glob, "/")
	if glob == "" {
		return nil, fmt.Errorf("空模式")
	}

	var b strings.Builder
	b.WriteString("^")

	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if ch == '*' {
			if i+1 < len(runes) && runes[i+1] == '*' {
				b.WriteString(".*")
				i++
				continue
			}
			b.WriteString(`[^/]*`)
			continue
		}
		if ch == '?' {
			b.WriteString(`[^/]`)
			continue
		}

		switch ch {
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteRune('\\')
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}

	b.WriteString("$")
	return regexp.Compile(b.String())
}

