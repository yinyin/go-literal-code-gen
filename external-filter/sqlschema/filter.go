package sqlschema

import (
	"log"
	"os"
	"regexp"
	"strconv"

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
	SymbolName       string
	MetaName         string
	Revision         int32
	Entry            *literalcodegen.LiteralEntry
	MigrationEntries []*literalcodegen.LiteralEntry
}

func newTablePropertyFromTitle1(entry *literalcodegen.LiteralEntry) (prop *tableProperty) {
	if "" == entry.TitleText {
		return nil
	}
	m := tablePropTitleTrap.FindStringSubmatchIndex(entry.TitleText)
	if nil == m {
		return nil
	}
	symbolName := entry.TitleText[m[2]:m[3]]
	metaName := entry.TitleText[m[4]:m[5]]
	revText := entry.TitleText[m[6]:m[7]]
	revValue, err := strconv.ParseInt(revText, 10, 31)
	if nil != err {
		log.Printf("WARN: cannot parse revision value: %v, %v: %v", entry.TitleText, revText, err)
		return nil
	}
	prop = &tableProperty{
		SymbolName:       symbolName,
		MetaName:         metaName,
		Revision:         int32(revValue),
		Entry:            entry,
		MigrationEntries: make([]*literalcodegen.LiteralEntry, revValue),
	}
	prop.initMigrationEntries()
	return prop
}

func (prop *tableProperty) feedMigrationEntries(entries []*literalcodegen.LiteralEntry) {
	for _, entry := range entries {
		if entry.TitleText == "" {
			continue
		}
		m := migrateRevTitleTrap.FindStringSubmatchIndex(entry.TitleText)
		if nil == m {
			continue
		}
		targetRevText := entry.TitleText[m[2]:m[3]]
		if targetRevValue, err := strconv.ParseInt(targetRevText, 10, 31); nil != err {
			log.Printf("WARN: failed on parsing revision value (%v): %v, %v: %v", prop.SymbolName, targetRevText, entry.TitleText, err)
			continue
		} else if (targetRevValue <= 0) || (int(targetRevValue) >= len(prop.MigrationEntries)) {
			log.Printf("WARN: revision out of boundary (%v): %v (max-rev=%v)", prop.SymbolName, targetRevValue, prop.Revision)
			continue
		} else {
			prop.MigrationEntries[int(targetRevValue)] = entry
		}
	}
}

func (prop *tableProperty) warnEmptyMigrationEntries() {
	for idx, targetRevEntry := range prop.MigrationEntries {
		if (0 == idx) && (targetRevEntry != nil) {
			log.Printf("WARN: target revision 0 with migration: %v, %v", prop.SymbolName, targetRevEntry)
		} else if (0 != idx) && (targetRevEntry == nil) {
			log.Printf("WARN: target revision %d with empty migration: %v, %v", idx, prop.SymbolName, targetRevEntry)
		}
	}
}

func (prop *tableProperty) initMigrationEntries() {
	for _, entry := range prop.Entry.ChildEntries {
		if entry.TitleText != "Migrations" {
			continue
		}
		prop.feedMigrationEntries(entry.ChildEntries)
	}
	prop.warnEmptyMigrationEntries()
}

func (prop *tableProperty) setupEntriesName() {
	prop.Entry.Name = "sqlCreate" + prop.SymbolName
	for idx, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		if entry.TranslationMode == literalcodegen.TranslateAsBuilder {
			entry.Name = "makeSQLMigrate"
		} else {
			entry.Name = "sqlMigrate"
		}
		entry.Name += prop.SymbolName + "ToRev" + strconv.FormatInt(int64(idx), 10)
	}
}

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
	MetaTableEntry *literalcodegen.LiteralEntry

	ParseRevisionCode  *literalcodegen.LiteralEntry
	FetchRevisionCode  *literalcodegen.LiteralEntry
	UpdateRevisionCode *literalcodegen.LiteralEntry

	TableProperties []*tableProperty
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
			if prop := newTablePropertyFromTitle1(entry); nil != prop {
				prop.setupEntriesName()
				filter.TableProperties = append(filter.TableProperties, prop)
			}
		}
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (filter *CodeGenerateFilter) GenerateExternalCode(fp *os.File, entries []*literalcodegen.LiteralEntry) (err error) {
	return nil
}
