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

func (prop *tableProperty) fetchRoutines() (revisionParseCodeText string, revisionFetchCodeTexts, revisionUpdateCodeTexts []string, err error) {
	var routineEntries []*literalcodegen.LiteralEntry
	for _, child := range prop.Entry.ChildEntries {
		if child.TitleText != "Routines" {
			continue
		}
		routineEntries = child.ChildEntries
	}
	if nil == routineEntries {
		return
	}
	for _, entry := range routineEntries {
		switch entry.TitleText {
		case "parse revision":
			if revisionParseCodeText, err = entry.FilteredContentLine(); nil != err {
				return
			}
		case "fetch revision":
			revisionFetchCodeTexts, err = entry.FilteredContent()
			if nil != err {
				return
			}
		case "update revision":
			revisionUpdateCodeTexts, err = entry.FilteredContent()
			if nil != err {
				return
			}
		default:
			log.Printf("WARN: (%v) unknown routine: %v", prop.Entry.TitleText, entry.TitleText)
		}
	}
	return
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
		} else if (targetRevValue <= 1) || (int(targetRevValue) > len(prop.MigrationEntries)) {
			log.Printf("WARN: revision out of boundary (%v): %v (max-rev=%v)", prop.SymbolName, targetRevValue, prop.Revision)
			continue
		} else {
			sourceRevValue := int(targetRevValue) - 1
			prop.MigrationEntries[sourceRevValue] = entry
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
	prop.Entry.Name = prop.sqlCreateSymbol()
	for idx, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		entry.Name = prop.migrateEntrySymbol(entry, int32(idx))
	}
}

func (prop *tableProperty) metaKeySymbol() string {
	return "metaKey" + prop.SymbolName + "SchemaRev"
}

func (prop *tableProperty) currentRevisionSymbol() string {
	return "current" + prop.SymbolName + "SchemaRev"
}

func (prop *tableProperty) sqlCreateSymbol() string {
	return "sqlCreate" + prop.SymbolName
}

func (prop *tableProperty) migrateEntrySymbol(entry *literalcodegen.LiteralEntry, sourceRev int32) string {
	var symbolPrefix string
	if entry.TranslationMode == literalcodegen.TranslateAsBuilder {
		symbolPrefix = "makeSQLMigrate"
	} else {
		symbolPrefix = "sqlMigrate"
	}
	return symbolPrefix + prop.SymbolName + "ToRev" + strconv.FormatInt(int64(sourceRev+1), 10)
}

func (prop *tableProperty) upgradeRoutineSymbol() string {
	return "upgradeSchema" + prop.SymbolName
}

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
	MetaTableEntry *literalcodegen.LiteralEntry

	ParseRevisionCodeText   string
	FetchRevisionCodeLines  []string
	UpdateRevisionCodeLines []string

	TableProperties []*tableProperty
}

// NewCodeGenerateFilter create an instance of CodeGenerateFilter
func NewCodeGenerateFilter() (filter *CodeGenerateFilter) {
	return &CodeGenerateFilter{}
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
			filter.MetaTableEntry = entry
		}
	}
	log.Printf("sql-schema filter: had %d table entries", len(filter.TableProperties))
	return nil
}

func (filter *CodeGenerateFilter) cacheBaseRoutines() (err error) {
	if len(filter.TableProperties) < 1 {
		return nil
	}
	metaTableProp := filter.TableProperties[0]
	filter.ParseRevisionCodeText, filter.FetchRevisionCodeLines, filter.UpdateRevisionCodeLines, err = metaTableProp.fetchRoutines()
	return err
}

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
		codeLine := "\tcase " + prop.metaKeySymbol() + ":\n" +
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
		"func (m *schemaManager) updateBaseTableSchemaRevision(key string, rev int32) (err error) {\n"); nil != err {
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
	if _, err = fp.WriteString("func (m *schemaManager) execBaseSchemaModification(sqlStmt, schemaMetaKey string, targetRev int32) (err error) {\n" +
		"\tif _, err = m.conn.Exec(sqlStmt); nil != err {\n" +
		"\t\treturn\n" +
		"\t}\n" +
		"\treturn m.updateBaseTableSchemaRevision(schemaMetaKey, targetRev)\n" +
		"}\n\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) generateBaseSchemaUpgradeRoutine(fp *os.File, prop *tableProperty) (err error) {
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.upgradeRoutineSymbol() + "(currentRev int32) (schemaChanged bool, err error) {\n" +
		"\tswitch currentRev {\n" +
		"\tcase " + prop.currentRevisionSymbol() + ":\n" +
		"\t\treturn false, nil\n" +
		"\tcase 0:\n" +
		"\t\tif err = m.execBaseSchemaModification(" + prop.sqlCreateSymbol() + ", " + prop.metaKeySymbol() + ", " + prop.currentRevisionSymbol() + "); nil == err {\n" +
		"\t\t\treturn true, nil\n" +
		"\t\t}\n"); nil != err {
		return
	}
	for sourceRev, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		if _, err = fp.WriteString("\tcase " + strconv.FormatInt(int64(sourceRev), 10) + ":\n" +
			"\t\tif err = m.execBaseSchemaModification(" + prop.migrateEntrySymbol(entry, int32(sourceRev)) + ", " + prop.metaKeySymbol() + ", " + prop.currentRevisionSymbol() + "); nil == err {\n" +
			"\t\t\treturn true, nil\n" +
			"\t\t}\n"); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\tdefault:\n" +
		"\t\terr = fmt.Errorf(\"unknown " + prop.MetaName + " schema revision: %d\", currentRev)\n" +
		"\t}\n" +
		"\treturn\n" +
		"}\n\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) generateSchemaUpgradeCodes(fp *os.File) (err error) {
	for _, prop := range filter.TableProperties {
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsConst:
			err = filter.generateBaseSchemaUpgradeRoutine(fp, prop)
		default:
			if _, err = fp.WriteString("// upgrade routine for symbol not generated: " + prop.SymbolName + "\n"); nil != err {
				return
			}
		}
		if nil != err {
			return
		}
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (filter *CodeGenerateFilter) GenerateExternalCode(fp *os.File, entries []*literalcodegen.LiteralEntry) (err error) {
	if err = filter.cacheBaseRoutines(); nil != err {
		return
	}
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
	if err = filter.generateSchemaUpgradeCodes(fp); nil != err {
		return
	}
	return nil
}
