package sqlschema

import (
	"log"
	"strconv"

	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

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

func (prop *tableProperty) fetchRoutines() (revisionFetchPrepareCodeTexts, revisionFetchCodeTexts, revisionUpdateCodeTexts []string, err error) {
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
		case "prepare fetch revision":
			if revisionFetchPrepareCodeTexts, err = entry.FilteredContent(); nil != err {
				return
			}
		case "fetch revision":
			if revisionFetchCodeTexts, err = entry.FilteredContent(); nil != err {
				return
			}
		case "update revision":
			if revisionUpdateCodeTexts, err = entry.FilteredContent(); nil != err {
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
		entry.PushDownReplaceRules()
		prop.feedMigrationEntries(entry.ChildEntries)
	}
	prop.warnEmptyMigrationEntries()
}

func (prop *tableProperty) updateMigrationEntryParameters(entry *literalcodegen.LiteralEntry) {
	if entry.TranslationMode == literalcodegen.TranslateAsBuilder {
		if ("" == entry.Name) && (0 == len(entry.Parameters)) {
			entry.Parameters = prop.Entry.Parameters
		}
	}
}

func (prop *tableProperty) setupEntriesPrototypes() {
	prop.Entry.Name = prop.sqlCreateSymbol()
	for idx, entry := range prop.MigrationEntries {
		if nil == entry {
			continue
		}
		prop.updateMigrationEntryParameters(entry)
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
	if prop.Entry.TranslationMode == literalcodegen.TranslateAsConst {
		return "UpgradeSchema" + prop.SymbolName
	}
	return "upgradeSchema" + prop.SymbolName
}

func (prop *tableProperty) upgradeWithRevisionRecordsRoutineSymbol() string {
	if prop.Entry.TranslationMode == literalcodegen.TranslateAsBuilder {
		return "UpgradeSchemaOf" + prop.SymbolName
	}
	return "upgradeSchema" + prop.SymbolName + "WithRevisions"
}

func (prop *tableProperty) execSchemaModificationSymbol() string {
	return "exec" + prop.SymbolName + "SchemaModification"
}

func (prop *tableProperty) isSchemasUpToDateSymbol() string {
	return "is" + prop.SymbolName + "SchemasUpToDate"
}

func (prop *tableProperty) schemaRevisionRecordStructSymbol() string {
	return "schemaRevisionOf" + prop.SymbolName
}

func (prop *tableProperty) fetchSchemaRevisionRecordsSymbol() string {
	return "fetchSchemaRevisionOf" + prop.SymbolName
}
