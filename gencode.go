package literalcodegen

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func generateHeadingCode(fp *os.File, entries []*LiteralEntry) (err error) {
	for _, entry := range entries {
		for _, line := range entry.Content {
			if _, err = fp.WriteString(line); nil != err {
				return err
			}
			uch := []rune(line)
			if '\n' != uch[len(uch)-1] {
				if _, err = fp.WriteString("\n"); nil != err {
					return err
				}
			}
		}
		if _, err = fp.WriteString("\n"); nil != err {
			return err
		}
	}
	return nil
}

func generateLiteralCodeAsConst(fp *os.File, entry *LiteralEntry) (err error) {
	codeLine := "const " + entry.Name + " = "
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	lastLineIndex := len(entry.Content) - 1
	for idx, line := range entry.Content {
		codeLine = strconv.Quote(line)
		if idx != 0 {
			codeLine = "\t\t" + codeLine
		}
		if idx != lastLineIndex {
			codeLine = codeLine + " +"
		}
		codeLine = codeLine + "\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	_, err = fp.WriteString("\n")
	return
}

func generateLiteralCodeAsBuilder(fp *os.File, entry *LiteralEntry) (err error) {
	codeLine := "func " + entry.Name + "(" + strings.Join(entry.Parameters, ", ") + ") string {\n" +
		"\treturn "
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	lastLineIndex := len(entry.Content) - 1
	for idx, line := range entry.Content {
		// TODO: process replacement
		codeLine = strconv.Quote(line)
		if idx != 0 {
			codeLine = "\t\t" + codeLine
		}
		if idx != lastLineIndex {
			codeLine = codeLine + " +"
		}
		codeLine = codeLine + "\n"
		if _, err = fp.WriteString(codeLine); nil != err {
			return
		}
	}
	_, err = fp.WriteString("}\n\n")
	return
}

func generateLiteralCodes(fp *os.File, entries []*LiteralEntry) (err error) {
	for _, entry := range entries {
		switch entry.TranslationMode {
		case TranslateAsConst:
			err = generateLiteralCodeAsConst(fp, entry)
		case TranslateAsBuilder:
			err = generateLiteralCodeAsBuilder(fp, entry)
		default:
			err = fmt.Errorf("unknown literal code generating mode: %d (%#v)", entry.TranslationMode, entry)
		}
		if nil != err {
			return
		}
	}
	return nil
}

// GenerateGoCodeFile generate code and save to given file path.
func GenerateGoCodeFile(path string, code *LiteralCode) (err error) {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if nil != err {
		return
	}
	defer fp.Close()
	if err = generateHeadingCode(fp, code.HeadingCodes); nil != err {
		return
	}
	return generateLiteralCodes(fp, code.LiteralConstants)
}
