package sqlschema

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

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

func parametersToArguments(params []string) (args []string) {
	for _, param := range params {
		aux := strings.SplitN(param, " ", 2)
		if len(aux) >= 1 {
			args = append(args, aux[0])
		}
	}
	return
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
	if _, err = fp.WriteString("\treturn true\n" +
		"}\n\n"); nil != err {
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
		if codeLine = strings.TrimSpace(codeLine); codeLine == "" {
			continue
		}
		if _, err = fp.WriteString(codeLine + "\n"); nil != err {
			return err
		}
	}
	if _, err = fp.WriteString("\tschemaRev = &schemaRevision{}\n" +
		"\tfor rows.Next() {\n" +
		"\t\tvar metaKey, metaValue string\n" +
		"\t\tif err = rows.Scan(&metaKey, &metaValue); nil != err {\n" +
		"\t\t\treturn nil, err\n" +
		"\t\t}\n" +
		"\t\tswitch metaKey {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		codeLine := "\t\tcase " + prop.metaKeySymbol() + ":\n" +
			"\t\t\tif schemaRev." + prop.SymbolName + ", err = " + filter.ParseRevisionCodeText + "(metaValue); nil != err {\n" +
			"\t\t\t\treturn nil, err\n" +
			"\t\t\t}\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\t\t}\n" +
		"\t}\n" +
		"\treturn schemaRev, nil\n" +
		"}\n\n" +
		"func (m *schemaManager) updateBaseTableSchemaRevision(key string, rev int32) (err error) {\n"); nil != err {
		return
	}
	for _, codeLine := range filter.UpdateRevisionCodeLines {
		if codeLine = strings.TrimSpace(codeLine); codeLine == "" {
			continue
		}
		if _, err = fp.WriteString(codeLine + "\n"); nil != err {
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

func (filter *CodeGenerateFilter) generateBuilderSchemaUpgradeRoutine(fp *os.File, prop *tableProperty) (err error) {
	// TODO: update revision
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.upgradeRoutineSymbol() + "(currentRev int32, " + strings.Join(prop.Entry.Parameters, ", ") + ") (schemaChanged bool, err error) {\n" +
		"\tswitch currentRev {\n" +
		"\tcase " + prop.currentRevisionSymbol() + ":\n" +
		"\t\treturn false, nil\n" +
		"\tcase 0:\n" +
		"\t\tif err = m.execBaseSchemaModification(" + prop.sqlCreateSymbol() + "(" + strings.Join(parametersToArguments(prop.Entry.Parameters), ", ") + ")" + ", " + prop.metaKeySymbol() + ", " + prop.currentRevisionSymbol() + "); nil == err {\n" +
		"\t\t\treturn true, nil\n" +
		"\t\t}\n"); nil != err {
		return
	}
	for sourceRev, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		if _, err = fp.WriteString("\tcase " + strconv.FormatInt(int64(sourceRev), 10) + ":\n" +
			"\t\tif err = m.execBaseSchemaModification(" + prop.migrateEntrySymbol(entry, int32(sourceRev)) + "(" + strings.Join(parametersToArguments(prop.Entry.Parameters), ", ") + ")" + ", " + prop.metaKeySymbol() + ", " + prop.currentRevisionSymbol() + "); nil == err {\n" +
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
		case literalcodegen.TranslateAsBuilder:
			err = filter.generateBuilderSchemaUpgradeRoutine(fp, prop)
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
