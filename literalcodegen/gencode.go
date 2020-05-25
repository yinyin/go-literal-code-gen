package literalcodegen

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func generatePassthroughGoCode(fp *os.File, entry *LiteralEntry) (err error) {
	for _, line := range entry.Content {
		if _, err = fp.WriteString(line); nil != err {
			return err
		}
		uch := []rune(line)
		lch := len(uch) - 1
		if (lch < 0) || ('\n' != uch[lch]) {
			if _, err = fp.WriteString("\n"); nil != err {
				return err
			}
		}
	}
	return nil
}

func generateHeadingCode(fp *os.File, entries []*LiteralEntry) (err error) {
	for _, entry := range entries {
		if err = generatePassthroughGoCode(fp, entry); nil != err {
			return
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
	content, err := entry.FilteredContent()
	if nil != err {
		return
	}
	lastLineIndex := len(content) - 1
	for idx, line := range content {
		if err = writeSimpleLiteralText(fp, line, idx, lastLineIndex); nil != err {
			return
		}
	}
	_, err = fp.WriteString("\n")
	return
}

func generateLiteralCodeAsBuilder(fp *os.File, entry *LiteralEntry) (err error) {
	var codeLine string
	codeLine = "func " + entry.Name + "(" + strings.Join(entry.Parameters, ", ") + ") string {\n"
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	if entry.BuilderPrepare != nil {
		if err = generatePassthroughGoCode(fp, entry.BuilderPrepare); nil != err {
			return
		}
	}
	codeLine = "\treturn "
	if _, err = fp.WriteString(codeLine); nil != err {
		return
	}
	content, err := entry.FilteredContent()
	if nil != err {
		return
	}
	lastLineIndex := len(content) - 1
	for idx, line := range content {
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
		if (entry.Name == "") || (entry.Name == "-") {
			log.Printf("skip: %v", entry.TitleText)
			continue
		}
		switch entry.TranslationMode {
		case TranslateAsNoop:
			err = nil
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
func GenerateGoCodeFile(path string, code *LiteralCode, externalFilter ExternalFilter) (err error) {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if nil != err {
		return
	}
	defer fp.Close()
	if err = generateHeadingCode(fp, code.HeadingCodes); nil != err {
		return
	}
	if nil != externalFilter {
		if err = externalFilter.PreCodeGenerate(code.LiteralConstants); nil != err {
			return
		}
	}
	if err = generateLiteralCodes(fp, code.LiteralConstants); nil != err {
		return
	}
	if nil != externalFilter {
		return externalFilter.GenerateExternalCode(fp, code.LiteralConstants)
	}
	return nil
}
