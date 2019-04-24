package literalcodegen

import (
	"log"
	"strings"
	"unicode"
)

const (
	// TranslateAsNoop set translation to no-op
	TranslateAsNoop = iota

	// TranslateAsConst set translation mode to constant
	TranslateAsConst

	// TranslateAsBuilder set translation mode to builder function
	TranslateAsBuilder
)

// LiteralEntry represent one literal entity to generate
type LiteralEntry struct {
	LevelDepth int
	TitleText  string
	Name       string
	Parameters []string

	TranslationMode       int
	TrimSpace             bool
	PreserveNewLine       bool
	TailNewLine           bool
	DisableLanguageFilter bool

	Content            []string
	LanguageType       string
	LanguageFilterArgs []string

	ParentEntry  *LiteralEntry
	ChildEntries []*LiteralEntry

	ExternalFilterData interface{}

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

func (entry *LiteralEntry) attachToParent(parent *LiteralEntry) {
	entry.TranslationMode = parent.TranslationMode
	entry.TrimSpace = parent.TrimSpace
	entry.PreserveNewLine = parent.PreserveNewLine
	entry.TailNewLine = parent.TailNewLine
	entry.DisableLanguageFilter = parent.DisableLanguageFilter
	entry.LevelDepth = parent.LevelDepth + 1
	entry.ParentEntry = parent
	parent.ChildEntries = append(parent.ChildEntries, entry)
}

// PushDownReplaceRules pushes replacing rules to children nodes without replacing rules
func (entry *LiteralEntry) PushDownReplaceRules() {
	localReplaceRules := entry.replaceRules
	if nil == localReplaceRules {
		return
	}
	for _, child := range entry.ChildEntries {
		if nil != child.replaceRules {
			continue
		}
		child.replaceRules = localReplaceRules
		child.PushDownReplaceRules()
	}
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
