package literalcodegen

import (
	"log"
	"strings"
	"unicode"
)

// TranslateAsConst set translation mode to constant
const TranslateAsConst = 1

// TranslateAsBuilder set translation mode to builder function
const TranslateAsBuilder = 2

// LiteralEntry represent one literal entity to generate
type LiteralEntry struct {
	Name                  string
	TranslationMode       int
	TrimSpace             bool
	PreserveNewLine       bool
	TailNewLine           bool
	DisableLanguageFilter bool
	Parameters            []string
	Content               []string
	LanguageType          string
	LanguageFilterArgs    []string

	replaceRules []*ReplaceRule
}

// NewLiteralEntry create a new instance of LiteralEntry and set properties to default values
func NewLiteralEntry() *LiteralEntry {
	return &LiteralEntry{}
}

// AppendContent add given content line by line and transform with specified configuration
func (entry *LiteralEntry) AppendContent(content, langType string, langFilterArgs []string) {
	lineBuffer := strings.Split(content, "\n")
	lastLineIndex := len(lineBuffer) - 1
	for idx, line := range lineBuffer {
		if entry.TrimSpace {
			line = strings.TrimSpace(line)
		} else {
			line = strings.TrimRightFunc(line, unicode.IsSpace)
		}
		if ((idx == lastLineIndex) && (entry.TailNewLine)) || entry.PreserveNewLine {
			line = line + "\n"
		} else if "" == line {
			continue
		}
		entry.Content = append(entry.Content, line)
	}
	if "" == entry.LanguageType {
		entry.LanguageType = langType
		entry.LanguageFilterArgs = langFilterArgs
	} else if nil != langFilterArgs {
		log.Printf("WARN: only filter arguments from first code block will be take: %q", langFilterArgs)
	}
}

func (entry *LiteralEntry) appendReplaceRule(rule *ReplaceRule) {
	entry.replaceRules = append(entry.replaceRules, rule)
}

// FilteredContent retun content filtered with language filter
func (entry *LiteralEntry) FilteredContent() (content []string, err error) {
	if entry.DisableLanguageFilter {
		return entry.Content, nil
	}
	return runLanaguageFilter(entry.LanguageType, entry.Content, entry.LanguageFilterArgs)
}

// LiteralCode represent one literal code module to generate
type LiteralCode struct {
	HeadingCodes     []*LiteralEntry
	LiteralConstants []*LiteralEntry
}

// NewHeadingCode allocate and append one literal entry as heading code node
func (l *LiteralCode) NewHeadingCode() (allocated *LiteralEntry) {
	allocated = NewLiteralEntry()
	l.HeadingCodes = append(l.HeadingCodes, allocated)
	return allocated
}

// NewLiteralConstant allocate and append one literal entry as literal constant node
func (l *LiteralCode) NewLiteralConstant() (allocated *LiteralEntry) {
	allocated = NewLiteralEntry()
	l.LiteralConstants = append(l.LiteralConstants, allocated)
	return allocated
}
