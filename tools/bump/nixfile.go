package main

import (
	"fmt"
	"regexp"

	"github.com/go-faster/errors"
)

// findBlock locates `name = { ... }` in src and returns the byte offsets of
// the opening and closing braces (inclusive), matching nested braces by depth.
func findBlock(src, name string) (openBrace, closeBrace int, err error) {
	re := regexp.MustCompile(regexp.QuoteMeta(name) + `\s*=\s*\{`)
	loc := re.FindStringIndex(src)
	if loc == nil {
		return 0, 0, errors.Errorf("variable %q not found", name)
	}

	openBrace = loc[1] - 1
	depth := 0
	for i := openBrace; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return openBrace, i, nil
			}
		}
	}
	return 0, 0, errors.Errorf("unbalanced braces for %q", name)
}

// getVersion reads a `varName = "value";` assignment.
func getVersion(src, varName string) (string, error) {
	re := regexp.MustCompile(regexp.QuoteMeta(varName) + `\s*=\s*"([^"]*)"\s*;`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		return "", errors.Errorf("variable %q not found", varName)
	}
	return m[1], nil
}

// setVersion rewrites a `varName = "value";` assignment in place.
func setVersion(src, varName, newVersion string) (string, error) {
	re := regexp.MustCompile(regexp.QuoteMeta(varName) + `\s*=\s*"[^"]*"\s*;`)
	if !re.MatchString(src) {
		return "", errors.Errorf("variable %q not found", varName)
	}
	return re.ReplaceAllString(src, fmt.Sprintf(`%s = "%s";`, varName, newVersion)), nil
}

// setHashForSystem rewrites the `hash = "...";` field of `tableVar.system`.
func setHashForSystem(src, tableVar, system, newHash string) (string, error) {
	tOpen, tClose, err := findBlock(src, tableVar)
	if err != nil {
		return "", err
	}
	table := src[tOpen : tClose+1]

	sOpen, sClose, err := findBlock(table, system)
	if err != nil {
		return "", errors.Wrapf(err, "system %q in %q", system, tableVar)
	}
	sub := table[sOpen : sClose+1]

	hashRe := regexp.MustCompile(`hash\s*=\s*"[^"]*"`)
	if !hashRe.MatchString(sub) {
		return "", errors.Errorf("no hash field in %s.%s", tableVar, system)
	}
	newSub := hashRe.ReplaceAllString(sub, fmt.Sprintf(`hash = "%s"`, newHash))

	newTable := table[:sOpen] + newSub + table[sClose+1:]
	return src[:tOpen] + newTable + src[tClose+1:], nil
}
