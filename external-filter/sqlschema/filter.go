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
	if tablePropTitleTrap, err = regexp.Compile("([a-zA-Z0-9_]+)\\s+\\(([a-zA-Z0-9-_]+)\\)\\s+r\\.\\s*([0-9]+)"); nil != err {
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

func (prop *tableProperty) metaKeySymbol() string {
	return "metaKey" + prop.SymbolName + "SchemaRev"
}

func (prop *tableProperty) currentRevisionSymbol() string {
	return "current" + prop.SymbolName + "SchemaRev"
}

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
	MetaTableEntry *literalcodegen.LiteralEntry

	ParseRevisionCode  *literalcodegen.LiteralEntry
	FetchRevisionCode  *literalcodegen.LiteralEntry
	UpdateRevisionCode *literalcodegen.LiteralEntry

	ParseRevisionCodeText   string
	FetchRevisionCodeLines  []string
	UpdateRevisionCodeLines []string

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

func (filter *CodeGenerateFilter) fetchParseRevisionCodeText() (err error) {
	codeLines, err := filter.ParseRevisionCode.FilteredContent()
	if nil != err {
		return
	}
	codeLineCount := len(codeLines)
	if 1 > codeLineCount {
		log.Printf("WARN: empty ParseRevisionCode")
		filter.ParseRevisionCodeText = ""
	} else {
		if codeLineCount > 1 {
			log.Printf("WARN: ParseRevisionCode has more than 1 lines, only first line will be use: %d", codeLineCount)
		}
		filter.ParseRevisionCodeText = codeLines[0]
	}
	return nil
}

func (filter *CodeGenerateFilter) fetchPredefinedCodeLines() (err error) {
	if err = filter.fetchParseRevisionCodeText(); nil != err {
		return
	}
	filter.FetchRevisionCodeLines, err = filter.FetchRevisionCode.FilteredContent()
	if nil != err {
		return
	}
	filter.UpdateRevisionCodeLines, err = filter.UpdateRevisionCode.FilteredContent()
	if nil != err {
		return
	}
	return nil
}

// PreCodeGenerate is invoked before literal code generation
func (filter *CodeGenerateFilter) PreCodeGenerate(entries []*literalcodegen.LiteralEntry) (err error) {
	if err = compileTrapRegexps(); nil != err {
		return
	}
	for _, entry := range entries {
		if 0 != entry.LevelDepth {
			continue
		}
		if prop := newTablePropertyFromTitle1(entry); nil != prop {
			prop.setupEntriesName()
			filter.TableProperties = append(filter.TableProperties, prop)
		} else {
			continue
		}
		if nil == filter.MetaTableEntry {
			filter.feedMetaTableEntry(entry)
		}
	}
	log.Printf("sql-schema filter: had %d table entries", len(filter.TableProperties))
	return nil
}

/*
const metaXRunMetaSchemaRev = "xrun-meta.schema"
const currentXRunMetaSchemaRev = 1

*/

func (filter *CodeGenerateFilter) generateSchemaRevisionConstant(fp *os.File) (err error) {
	for _, prop := range filter.TableProperties {
		codeLine := "const " + prop.metaKeySymbol() + " = " + strconv.Quote(prop.MetaName+".schema") + "\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		codeLine := "const " + prop.currentRevisionSymbol() + " = " + strconv.FormatInt(int64(prop.Revision), 10) + "\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) generateSchemaRevisionStruct(fp *os.File) (err error) {
	if _, err = fp.WriteString("type schemaRevision struct {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		codeLine := "\t" + prop.SymbolName + " int32\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("}\n\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("func (rev *schemaRevision) IsUpToDate() bool {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		codeLine := "\tif " + prop.currentRevisionSymbol() + " != rev." + prop.SymbolName + " {\n" +
			"\t\treturn false\n" +
			"\t}\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\treturn true\n}\n\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) generateSchemaManager(fp *os.File) (err error) {
	if _, err = fp.WriteString("type schemaManager struct {\n" +
		"\tconn *sql.DB\n" +
		"}\n\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("func (m *schemaManager) FetchSchemaRevision() (schemaRev *schemaRevision, err error) {\n"); nil != err {
		return
	}
	for _, codeLine := range filter.FetchRevisionCodeLines {
		if _, err = fp.WriteString(codeLine); nil != err {
			return err
		}
	}
	if _, err = fp.WriteString("\tschemaRev = &schemaRevision{}\n" +
		"\tfor rows.Next() {\n" +
		"\tvar metaKey, metaValue string\n" +
		"\tif err = rows.Scan(&metaKey, &metaValue); nil != err {\n" +
		"\t\treturn nil, err\n" +
		"\t}\n" +
		"\tswitch metaKey {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		codeLine := "\tcase" + prop.metaKeySymbol() + ":\n" +
			"\t\tif schemaRev." + prop.SymbolName + ", err = " + filter.ParseRevisionCodeText + "(metaValue); nil != err {\n" +
			"\t\t\treturn nil, err\n" +
			"\t\t}\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\t}\n" +
		"\treturn schemaRev, nil\n" +
		"}\n\n" +
		"func (m *schemaManager) updateTableSchemaRevision(key string, rev int32) (err error) {"); nil != err {
		return
	}
	for _, codeLine := range filter.UpdateRevisionCodeLines {
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\treturn\n" +
		"}\n\n"); nil != err {
		return
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (filter *CodeGenerateFilter) GenerateExternalCode(fp *os.File, entries []*literalcodegen.LiteralEntry) (err error) {
	filter.fetchPredefinedCodeLines()
	if _, err = fp.WriteString("// ** SQL schema external filter\n\n"); nil != err {
		return
	}
	if err = filter.generateSchemaRevisionConstant(fp); nil != err {
		return
	}
	if err = filter.generateSchemaRevisionStruct(fp); nil != err {
		return
	}
	if err = filter.generateSchemaManager(fp); nil != err {
		return
	}
	return nil
}
