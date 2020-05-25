package literalcodegen

import (
	"log"
	"strings"
	"unicode"
)

// TranslationModeType represent type of translation mode.
type TranslationModeType int

const (
	// TranslateAsNoop set translation to no-op
	TranslateAsNoop TranslationModeType = iota

	// TranslateAsConst set translation mode to constant
	TranslateAsConst

	// TranslateAsBuilder set translation mode to builder function
	TranslateAsBuilder
)

// SubWorkType represent type of sub-works.
// SubWork is associated code such as preparing part for builder.
type SubWorkType int

const (
	// NotSubWork indicate given entry is not a sub-work.
	NotSubWork SubWorkType = iota

	// SubWorkBuilderPrepare indicate given entry is code block for builder prepare.
	SubWorkBuilderPrepare
)

// LiteralEntry represent one literal entity to generate
type LiteralEntry struct {
	LevelDepth int
	TitleText  string
	Name       string
	Parameters []string
	SubWork    SubWorkType

	TranslationMode       TranslationModeType
	TrimSpace             bool
	PreserveNewLine       bool
	KeepEmptyLine         bool
	TailNewLine           bool
	DisableLanguageFilter bool

	Content            []string
	LanguageType       string
	LanguageFilterArgs []string

	BuilderPrepare *LiteralEntry

	ParentEntry  *LiteralEntry
	ChildEntries []*LiteralEntry

	ExternalFilterData interface{}

	replaceRules []*ReplaceRule
}

// NewLiteralEntry create a new instance of LiteralEntry and set properties to default values
func NewLiteralEntry() *LiteralEntry {
	return &LiteralEntry{}
}

// GetBuilderPrepareNode return a builder prepare node.
// Existed one will be return if such node existed.
func (entry *LiteralEntry) GetBuilderPrepareNode() *LiteralEntry {
	if entry.BuilderPrepare == nil {
		entry.BuilderPrepare = &LiteralEntry{
			SubWork:     SubWorkBuilderPrepare,
			ParentEntry: entry,
		}
	}
	return entry.BuilderPrepare
}

// AppendContent add given content line by line and transform with specified configuration
func (entry *LiteralEntry) AppendContent(content, langType string, langFilterArgs []string) {
	if entry.KeepEmptyLine {
		content = strings.TrimRightFunc(content, unicode.IsSpace)
	}
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
		} else if ("" == line) && (!entry.KeepEmptyLine) {
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

// FilteredContent return content filtered with language filter
func (entry *LiteralEntry) FilteredContent() (content []string, err error) {
	if entry.DisableLanguageFilter {
		return entry.Content, nil
	}
	return runLanaguageFilter(entry.LanguageType, entry.Content, entry.LanguageFilterArgs)
}

// FilteredContentLine return first line from filtered content
func (entry *LiteralEntry) FilteredContentLine() (contentLine string, err error) {
	content, err := entry.FilteredContent()
	if nil != err {
		return
	}
	contentCount := len(content)
	if 1 > contentCount {
		log.Printf("WARN (FilteredContentLine): empty %v", entry.TitleText)
		return "", nil
	}
	if contentCount > 1 {
		log.Printf("WARN (FilteredContentLine): entry %v has more than 1 line, only first line will be return: %d", entry.TitleText, contentCount)
	}
	return content[0], nil
}

func (entry *LiteralEntry) attachToParent(parent *LiteralEntry) {
	entry.TranslationMode = parent.TranslationMode
	entry.TrimSpace = parent.TrimSpace
	entry.PreserveNewLine = parent.PreserveNewLine
	entry.KeepEmptyLine = parent.KeepEmptyLine
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
