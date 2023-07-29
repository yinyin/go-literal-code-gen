package sqlschema

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	interpolatetext "github.com/yinyin/go-interpolatetext"

	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

var tablePropTitleTrap *regexp.Regexp
var migrateRevTitleTrap *regexp.Regexp

func compileTrapRegexps() (err error) {
	if (nil != tablePropTitleTrap) && (nil != migrateRevTitleTrap) {
		return nil
	}
	if tablePropTitleTrap, err = regexp.Compile(`([a-zA-Z0-9_]+)\s+\(([a-zA-Z0-9-_]+)\)\s+r\.\s*([0-9]+)`); nil != err {
		log.Printf("ERR: failed on compiling regular expression for trapping table property: %v", err)
		return
	}
	if migrateRevTitleTrap, err = regexp.Compile(`To\s+r\.\s*([0-9]+)`); nil != err {
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

func writeTrimmedCodeLine(fp *os.File, codeLine string) (err error) {
	if codeLine = strings.TrimSpace(codeLine); codeLine == "" {
		return nil
	}
	_, err = fp.WriteString(codeLine + "\n")
	return
}

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
	MetaTableEntry *literalcodegen.LiteralEntry

	FetchRevisionPrepareCodeLines []string
	FetchRevisionCodeLines        []string
	UpdateRevisionCodeLines       []string

	FetchRevisionCodeTpl interpolatetext.TextMapInterpolationSlice

	TableProperties []*tableProperty

	GeneratedTODOs int
}

// NewCodeGenerateFilter create an instance of CodeGenerateFilter
func NewCodeGenerateFilter() (filter *CodeGenerateFilter) {
	return &CodeGenerateFilter{}
}

func (filter *CodeGenerateFilter) increaseTODOCount() {
	filter.GeneratedTODOs++
}

// PreCodeGenerate is invoked before literal code generation
func (filter *CodeGenerateFilter) PreCodeGenerate(entries []*literalcodegen.LiteralEntry) (err error) {
	if err = compileTrapRegexps(); nil != err {
		return
	}
	for _, entry := range entries {
		if entry.LevelDepth != 0 {
			continue
		}
		if prop := newTablePropertyFromTitle1(entry); nil != prop {
			prop.setupEntriesPrototypes()
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
	if filter.FetchRevisionPrepareCodeLines, filter.FetchRevisionCodeLines, filter.UpdateRevisionCodeLines, err = metaTableProp.fetchRoutines(); nil != err {
		return err
	}
	filter.FetchRevisionCodeTpl, err = interpolatetext.NewTextMapInterpolationSlice(filter.FetchRevisionCodeLines)
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
	for _, prop := range filter.TableProperties {
		if prop.Entry.TranslationMode != literalcodegen.TranslateAsBuilder {
			continue
		}
		if _, err = fp.WriteString("func " + prop.isSchemasUpToDateSymbol() + "(revRecords []*" + prop.schemaRevisionRecordStructSymbol() + ") bool {\n" +
			"\tfor _, recRec := range revRecords {\n" +
			"\t\tif " + prop.currentRevisionSymbol() + " != recRec.currentRev {\n" +
			"\t\t\treturn false\n" +
			"\t\t}\n" +
			"\t}\n" +
			"\treturn true\n" +
			"}\n\n"); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("type schemaRevision struct {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		var codeLine string
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsExplicitNoop:
			codeLine = ""
		case literalcodegen.TranslateAsConst:
			codeLine = "\t" + prop.SymbolName + " int32\n"
		case literalcodegen.TranslateAsBuilder:
			codeLine = "\t" + prop.SymbolName + " []*" + prop.schemaRevisionRecordStructSymbol() + "\n"
		default:
			codeLine = fmt.Sprintf("\t// TODO: unknown translation mode %v for symbol [%v]\n", prop.Entry.TranslationMode, prop.SymbolName)
		}
		if codeLine != "" {
			if _, err = fp.WriteString(codeLine); nil != err {
				return
			}
		}
	}
	if _, err = fp.WriteString("}\n\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("func (rev *schemaRevision) IsUpToDate() bool {\n"); nil != err {
		return
	}
	for _, prop := range filter.TableProperties {
		var codeLine string
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsConst:
			codeLine = "\tif " + prop.currentRevisionSymbol() + " != rev." + prop.SymbolName + " {\n" +
				"\t\treturn false\n" +
				"\t}\n"
		case literalcodegen.TranslateAsBuilder:
			codeLine = "\tif !" + prop.isSchemasUpToDateSymbol() + "(rev." + prop.SymbolName + ") {\n" +
				"\t\treturn false\n" +
				"\t}\n"
		}
		if codeLine != "" {
			if _, err = fp.WriteString(codeLine); nil != err {
				return
			}
		}
	}
	if _, err = fp.WriteString("\treturn true\n" +
		"}\n\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) hasConstTableProperty() bool {
	for _, prop := range filter.TableProperties {
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsConst:
			return true
		}
	}
	return false
}

func (filter *CodeGenerateFilter) generateSchemaManager(fp *os.File) (err error) {
	if _, err = fp.WriteString("type schemaManager struct {\n" +
		"\treferenceTableName string\n" +
		"\tctx context.Context\n" +
		"\tconn *sql.DB\n" +
		"}\n\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("func (m *schemaManager) FetchSchemaRevision() (schemaRev *schemaRevision, err error) {\n"); nil != err {
		return
	}
	if filter.hasConstTableProperty() {
		for _, codeLine := range filter.FetchRevisionPrepareCodeLines {
			if err = writeTrimmedCodeLine(fp, codeLine); nil != err {
				return
			}
		}
		if _, err = fp.WriteString("\tschemaRev = &schemaRevision{}\n"); nil != err {
			return
		}
		for _, prop := range filter.TableProperties {
			if prop.Entry.TranslationMode == literalcodegen.TranslateAsConst {
				textMap := map[string]string{
					"SCHEMA_REV_KEY": prop.metaKeySymbol(),
					"SCHEMA_REV_VAR": "schemaRev." + prop.SymbolName,
				}
				var codeLines []string
				if codeLines, err = filter.FetchRevisionCodeTpl.Apply(textMap, true); nil != err {
					return
				}
				for _, codeLine := range codeLines {
					if err = writeTrimmedCodeLine(fp, codeLine); nil != err {
						return
					}
				}
			}
		}
	} else {
		if _, err = fp.WriteString("\tschemaRev = &schemaRevision{}\n"); nil != err {
			return
		}
	}
	for _, prop := range filter.TableProperties {
		var codeLine string
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsBuilder:
			codeLine = "\tif schemaRev." + prop.SymbolName + ", err = m." + prop.fetchSchemaRevisionRecordsSymbol() + "(); nil != err {\n" +
				"\t\treturn nil, err\n" +
				"\t}\n"
		}
		if codeLine != "" {
			if _, err = fp.WriteString(codeLine); nil != err {
				return
			}
		}
	}
	if _, err = fp.WriteString("\treturn schemaRev, nil\n" +
		"}\n\n"); nil != err {
		return
	}
	if filter.hasConstTableProperty() {
		if _, err = fp.WriteString("func (m *schemaManager) updateBaseTableSchemaRevision(key string, rev int32) (err error) {\n"); nil != err {
			return
		}
		for _, codeLine := range filter.UpdateRevisionCodeLines {
			if err = writeTrimmedCodeLine(fp, codeLine); nil != err {
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
	}
	for _, prop := range filter.TableProperties {
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsBuilder:
			if err = filter.generateBuilderSchemaRevisionStructure(fp, prop); nil != err {
				return
			}
			if err = filter.generateBuilderFetchSchemaRevisionRoutine(fp, prop); nil != err {
				return
			}
			if err = filter.generateBuilderExecSchemaModificationRoutine(fp, prop); nil != err {
				return
			}
		}
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
	boundRev := len(prop.MigrationEntries) - 1
	for sourceRev, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		if _, err = fp.WriteString("\tcase " + strconv.FormatInt(int64(sourceRev), 10) + ":\n" +
			"\t\tif err = m.execBaseSchemaModification(" + prop.migrateEntrySymbol(entry, int32(sourceRev)) + ", " + prop.metaKeySymbol() + ", " + strconv.FormatInt(int64(sourceRev+1), 10) + "); nil == err {\n" +
			"\t\t\tschemaChanged = true\n"); nil != err {
			return
		}
		if boundRev != sourceRev {
			if _, err = fp.WriteString("\t\t} else {\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t\tfallthrough\n"); nil != err {
				return
			}
		} else {
			if _, err = fp.WriteString("\t\t}\n" +
				"\t\treturn\n"); nil != err {
				return
			}
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

func (filter *CodeGenerateFilter) generateBuilderExecSchemaModificationRoutine(fp *os.File, prop *tableProperty) (err error) {
	_, _, revisionUpdateCodeTexts, err := prop.fetchRoutines()
	if nil != err {
		return
	}
	if len(revisionUpdateCodeTexts) == 0 {
		revisionUpdateCodeTexts = []string{
			"\t// TODO: revision update code (Routines > update revision) will placed here",
		}
		filter.increaseTODOCount()
	}
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.updateSchemaRevisionSymbol() + "(" + strings.Join(prop.Entry.Parameters, ", ") + ", targetRev int32) (err error) {\n"); nil != err {
		return
	}
	for _, codeLine := range revisionUpdateCodeTexts {
		if err = writeTrimmedCodeLine(fp, codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\treturn\n" +
		"}\n\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.execSchemaModificationSymbol() + "(sqlStmt string, " + strings.Join(prop.Entry.Parameters, ", ") + ", targetRev int32) (err error) {\n" +
		"\tif _, err = m.conn.ExecContext(m.ctx, sqlStmt); nil != err {\n" +
		"\t\treturn\n" +
		"\t}\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("\terr = m." + prop.updateSchemaRevisionSymbol() + "(" + strings.Join(parametersToArguments(prop.Entry.Parameters), ", ") + ", targetRev)\n"); nil != err {
		return
	}
	if _, err = fp.WriteString("\treturn\n" +
		"}\n\n"); nil != err {
		return
	}
	return nil
}

func (filter *CodeGenerateFilter) generateBuilderSchemaUpgradeWithRevisionRecordsRoutine(fp *os.File, prop *tableProperty) (err error) {
	var params []string
	for _, param := range parametersToArguments(prop.Entry.Parameters) {
		params = append(params, "revRec."+param)
	}
	paramAsArgs := strings.Join(params, ", ")
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.upgradeWithRevisionRecordsRoutineSymbol() + "(revisionRecords []*" + prop.schemaRevisionRecordStructSymbol() + ") (schemaChanged bool, err error) {\n" +
		"\tfor _, revRec := range revisionRecords {\n" +
		"\t\tif changed, err := m." + prop.upgradeRoutineSymbol() + "(revRec.currentRev, " + paramAsArgs + "); nil != err {\n" +
		"\t\t\treturn schemaChanged, fmt.Errorf(\"upgrade " + prop.SymbolName + " failed (%#v): %#v\", revRec, err)\n" +
		"\t\t} else if changed {\n" +
		"\t\t\tschemaChanged = true\n" +
		"\t\t}\n" +
		"\t}\n" +
		"\treturn schemaChanged, nil\n" +
		"}\n\n"); nil != err {
		return
	}
	return
}

func (filter *CodeGenerateFilter) generateBuilderSchemaUpgradeRoutine(fp *os.File, prop *tableProperty) (err error) {
	paramAsArgs := strings.Join(parametersToArguments(prop.Entry.Parameters), ", ")
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.upgradeRoutineSymbol() + "(currentRev int32, " + strings.Join(prop.Entry.Parameters, ", ") + ") (schemaChanged bool, err error) {\n" +
		"\tswitch currentRev {\n" +
		"\tcase " + prop.currentRevisionSymbol() + ":\n" +
		"\t\treturn false, nil\n" +
		"\tcase 0:\n" +
		"\t\tif err = m." + prop.execSchemaModificationSymbol() + "(" + prop.sqlCreateSymbol() + "(" + paramAsArgs + ")" + ", " + paramAsArgs + ", " + prop.currentRevisionSymbol() + "); nil == err {\n" +
		"\t\t\treturn true, nil\n" +
		"\t\t}\n"); nil != err {
		return
	}
	boundRev := len(prop.MigrationEntries) - 1
	for sourceRev, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		var schemaUpdateInvokeLeadingCode string
		if migrateEntrySymbol := prop.migrateEntrySymbol(entry, int32(sourceRev)); migrateEntrySymbol != "" {
			schemaUpdateInvokeLeadingCode = prop.execSchemaModificationSymbol() + "(" + migrateEntrySymbol + "(" + paramAsArgs + "), "
		} else {
			schemaUpdateInvokeLeadingCode = prop.updateSchemaRevisionSymbol() + "("
		}
		if _, err = fp.WriteString("\tcase " + strconv.FormatInt(int64(sourceRev), 10) + ":\n" +
			"\t\tif err = m." + schemaUpdateInvokeLeadingCode + paramAsArgs + ", " + strconv.FormatInt(int64(sourceRev+1), 10) + "); nil == err {\n" +
			"\t\t\tschemaChanged = true\n"); nil != err {
			return
		}
		if boundRev != sourceRev {
			if _, err = fp.WriteString("\t\t} else {\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t\tfallthrough\n"); nil != err {
				return
			}
		} else {
			if _, err = fp.WriteString("\t\t}\n" +
				"\t\treturn\n"); nil != err {
				return
			}
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

func (filter *CodeGenerateFilter) generateBuilderSchemaRevisionStructure(fp *os.File, prop *tableProperty) (err error) {
	if _, err = fp.WriteString("type " + prop.schemaRevisionRecordStructSymbol() + " struct {\n" +
		"\tcurrentRev int32\n"); nil != err {
		return
	}
	for _, param := range prop.Entry.Parameters {
		if _, err = fp.WriteString("\t" + param + "\n"); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("}\n\n"); nil != err {
		return
	}
	return
}

func (filter *CodeGenerateFilter) generateBuilderFetchSchemaRevisionRoutine(fp *os.File, prop *tableProperty) (err error) {
	if _, err = fp.WriteString("func (m *schemaManager) " + prop.fetchSchemaRevisionRecordsSymbol() + "() (revisionRecords []*" + prop.schemaRevisionRecordStructSymbol() + ", err error) {\n"); nil != err {
		return
	}
	_, revisionFetchCodeTexts, _, err := prop.fetchRoutines()
	if nil != err {
		return
	}
	if len(revisionFetchCodeTexts) == 0 {
		revisionFetchCodeTexts = []string{
			"// TODO: revision fetch code (Routines > fetch revision) will placed here",
		}
		filter.increaseTODOCount()
	}
	for _, codeLine := range revisionFetchCodeTexts {
		if err = writeTrimmedCodeLine(fp, codeLine); nil != err {
			return
		}
	}
	if _, err = fp.WriteString("\treturn\n" +
		"}\n\n"); nil != err {
		return
	}
	return
}

func (filter *CodeGenerateFilter) generateSchemaUpgradeCodes(fp *os.File) (err error) {
	for _, prop := range filter.TableProperties {
		switch prop.Entry.TranslationMode {
		case literalcodegen.TranslateAsConst:
			err = filter.generateBaseSchemaUpgradeRoutine(fp, prop)
		case literalcodegen.TranslateAsBuilder:
			if err = filter.generateBuilderSchemaUpgradeRoutine(fp, prop); nil != err {
				return
			}
			err = filter.generateBuilderSchemaUpgradeWithRevisionRecordsRoutine(fp, prop)
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
	if _, err = fp.WriteString("\n" + fmt.Sprintf("// ** Generated code for %d table entries\n", len(filter.TableProperties))); nil != err {
		return
	}
	if filter.GeneratedTODOs > 0 {
		if _, err = fp.WriteString(fmt.Sprintf("// There are %d TODO tag(s) generated. Pleace fulfill missing code before proceed.", filter.GeneratedTODOs)); nil != err {
			return
		}
	}
	return nil
}
