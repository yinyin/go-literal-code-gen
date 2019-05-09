package literalcodegen

import (
	"strings"
	"unicode"
)

type languageContentFilter func(codeContent, filterArgs []string) (result []string, err error)

func parseSQLFilterArgs(filterArgs []string) (removeComments bool) {
	removeComments = true
	for _, arg := range filterArgs {
		if arg == "keep-comment" {
			removeComments = false
		}
	}
	return
}

func sqlContentFilter(codeContent, filterArgs []string) (result []string, err error) {
	removeComments := parseSQLFilterArgs(filterArgs)
	lastLineIndex := len(codeContent) - 1
	notNeedSpace := true
	for idx, line := range codeContent {
		if removeComments {
			stripped := strings.TrimLeftFunc(line, unicode.IsSpace)
			if strings.HasPrefix(stripped, "--") || strings.HasPrefix(stripped, "/*") {
				continue
			}
		}
		if idx == lastLineIndex {
			line = strings.TrimRightFunc(line, func(r rune) bool {
				if (r == ';') || unicode.IsSpace(r) {
					return true
				}
				return false
			})
		}
		if ch := []rune(line); len(ch) > 0 {
			firstCh := ch[0]
			lastCh := ch[len(ch)-1]
			if (!notNeedSpace) && (firstCh >= 'A') && (firstCh <= 'Z') {
				line = " " + line
			}
			notNeedSpace = ((lastCh == '(') || (lastCh == ','))
		}
		result = append(result, line)
	}
	return
}

func runLanaguageFilter(langType string, codeContent, filterArgs []string) (result []string, err error) {
	var filterCallable languageContentFilter
	switch langType {
	case "sql":
		filterCallable = sqlContentFilter
	}
	if nil == filterCallable {
		return codeContent, nil
	}
	return filterCallable(codeContent, filterArgs)
}
