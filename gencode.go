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

func writeSimpleLiteralText(fp *os.File, line string, currentLineIndex, lastLineIndex int) (err error) {
	codeLine := strconv.Quote(line)
	if currentLineIndex != 0 {
		codeLine = "\t\t" + codeLine
	}
	if currentLineIndex != lastLineIndex {
		codeLine = codeLine + " +"
	}
	codeLine = codeLine + "\n"
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	return
}

func appendLiteralText(codeLine, literalText string, hasCode bool) (string, bool) {
	if "" == literalText {
		return codeLine, hasCode
	}
	if hasCode {
		codeLine = codeLine + " + "
	}
	codeLine = codeLine + strconv.Quote(literalText)
	return codeLine, true
}

func appendLiteralCode(codeLine, codeText string, hasCode bool) (string, bool) {
	if "" == codeText {
		return codeLine, hasCode
	}
	if hasCode {
		codeLine = codeLine + " + "
	}
	codeLine = codeLine + "(" + codeText + ")"
	return codeLine, true
}

func writeReplacedLiteralCode(fp *os.File, lineSegs []*ReplaceResult, currentLineIndex, lastLineIndex int) (err error) {
	var codeLine string
	if currentLineIndex != 0 {
		codeLine = "\t\t"
	}
	hasCode := false
	for _, lineSeg := range lineSegs {
		codeLine, hasCode = appendLiteralText(codeLine, lineSeg.PrefixLiteral, hasCode)
		codeLine, hasCode = appendLiteralCode(codeLine, lineSeg.ReplacedCode, hasCode)
		codeLine, hasCode = appendLiteralText(codeLine, lineSeg.SuffixLiteral, hasCode)
	}
	if !hasCode {
		return
	}
	if currentLineIndex != lastLineIndex {
		codeLine = codeLine + " +"
	}
	codeLine = codeLine + "\n"
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	return
}

func generateLiteralCodeAsConst(fp *os.File, entry *LiteralEntry) (err error) {
	codeLine := "const " + entry.Name + " = "
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	lastLineIndex := len(entry.Content) - 1
	for idx, line := range entry.Content {
		if err = writeSimpleLiteralText(fp, line, idx, lastLineIndex); nil != err {
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
		replaced, err := doReplace(entry.replaceRules, line)
		if nil != err {
			return err
		}
		if nil == replaced {
			if err = writeSimpleLiteralText(fp, line, idx, lastLineIndex); nil != err {
				return err
			}
		} else {
			if err = writeReplacedLiteralCode(fp, replaced, idx, lastLineIndex); nil != err {
				return err
			}
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
