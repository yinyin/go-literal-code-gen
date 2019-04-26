package sqlschema

import (
	"log"
	"os"
	"regexp"

	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

var tablePropTitleTrap *regexp.Regexp
var migrateRevTitleTrap *regexp.Regexp

func compileTrapRegexps() (err error) {
	if (nil != tablePropTitleTrap) && (nil != migrateRevTitleTrap) {
		return nil
	}
	if tablePropTitleTrap, err = regexp.Compile("([a-zA-Z0-9_]+)\\s+\\(([a-zA-Z0-9_]+)\\)\\s+r\\.\\s*([0-9]+)"); nil != err {
		log.Printf("ERR: failed on compiling regular expression for trapping table property: %v", err)
		return
	}
	if migrateRevTitleTrap, err = regexp.Compile("To\\s+r\\.\\s*([0-9]+)"); nil != err {
		log.Printf("ERR: failed on compiling regular expression for trapping migration revision: %v", err)
		return
	}
	return
}

type tableProperty struct {
	SymbolName string
	MetaName   string
	Revision   int32
	Entry      *literalcodegen.LiteralEntry
}

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
	MetaTableEntry *literalcodegen.LiteralEntry

	ParseRevisionCode  *literalcodegen.LiteralEntry
	FetchRevisionCode  *literalcodegen.LiteralEntry
	UpdateRevisionCode *literalcodegen.LiteralEntry
}

// NewCodeGenerateFilter create an instance of CodeGenerateFilter
func NewCodeGenerateFilter() (filter *CodeGenerateFilter) {
	return &CodeGenerateFilter{}
}

func (filter *CodeGenerateFilter) feedMetaRoutines(entries []*literalcodegen.LiteralEntry) {
	for _, entry := range entries {
		switch entry.TitleText {
		case "parse revision":
			if nil != filter.ParseRevisionCode {
				log.Printf("WARN: over written existed ParseRevisionCode %v <= %v", filter.ParseRevisionCode, entry)
			}
			filter.ParseRevisionCode = entry
		case "fetch revision":
			if nil != filter.FetchRevisionCode {
				log.Printf("WARN: over written existed FetchRevisionCode %v <= %v", filter.FetchRevisionCode, entry)
			}
			filter.FetchRevisionCode = entry
		case "update revision":
			if nil != filter.UpdateRevisionCode {
				log.Printf("WARN: over written existed UpdateRevisionCode %v <= %v", filter.UpdateRevisionCode, entry)
			}
			filter.UpdateRevisionCode = entry
		default:
			log.Printf("WARN: unknown meta routine: %v", entry.TitleText)
		}
	}
}

func (filter *CodeGenerateFilter) feedMetaTableEntry(entry *literalcodegen.LiteralEntry) {
	filter.MetaTableEntry = entry
	for _, child := range entry.ChildEntries {
		if child.TitleText == "Routines" {
			filter.feedMetaRoutines(child.ChildEntries)
		}
	}
}

// PreCodeGenerate is invoked before literal code generation
func (filter *CodeGenerateFilter) PreCodeGenerate(entries []*literalcodegen.LiteralEntry) (err error) {
	if err = compileTrapRegexps(); nil != err {
		return
	}
	for _, entry := range entries {
		if 0 == entry.LevelDepth {
			if nil == filter.MetaTableEntry {
				filter.feedMetaTableEntry(entry)
			}
		}
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (filter *CodeGenerateFilter) GenerateExternalCode(fp *os.File, entries []*literalcodegen.LiteralEntry) (err error) {
	return nil
}
