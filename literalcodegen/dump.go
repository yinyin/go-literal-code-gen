package literalcodegen

import (
	"log"

	"gitlab.com/golang-commonmark/markdown"
)

func logLiteralEntry(entry *LiteralEntry) {
	log.Printf("- %s (depth=%d; mode=%d; subwork=%d): trim-space=%v, preserve-new-line=%v, tail-new-line=%v",
		entry.Name,
		entry.LevelDepth,
		entry.TranslationMode,
		entry.SubWork,
		entry.TrimSpace,
		entry.PreserveNewLine,
		entry.TailNewLine)
	log.Printf("  > param (%d):", len(entry.Parameters))
	for idx, param := range entry.Parameters {
		log.Printf("   %d: %v", idx, param)
	}
	log.Printf("  > content (%d):", len(entry.Content))
	for idx, val := range entry.Content {
		log.Printf("   %03d: %v", idx, val)
	}
	log.Printf("  > replace (%d):", len(entry.replaceRules))
	for idx, rule := range entry.replaceRules {
		log.Printf("   %d: %#v", idx, rule)
	}
	if entry.BuilderPrepare != nil {
		log.Printf("  > builder-prepare (%v) ||", entry.BuilderPrepare)
		logLiteralEntry(entry.BuilderPrepare)
	}
}

func logLiteralEntries(entries []*LiteralEntry) {
	if 0 == len(entries) {
		return
	}
	for idx, entry := range entries {
		log.Printf("* %d:", idx)
		logLiteralEntry(entry)
	}
}

// LogLiteralCode dump literal code object to log
func LogLiteralCode(code *LiteralCode) {
	log.Printf("# Heading Code (%d)", len(code.HeadingCodes))
	logLiteralEntries(code.HeadingCodes)
	log.Printf("# Literal Constants (%d)", len(code.LiteralConstants))
	logLiteralEntries(code.LiteralConstants)
}

func logMarkdownAST(tokens []markdown.Token) {
	spaceText := "                "
	for idx, tok := range tokens {
		var indentText string
		if lvl := tok.Level(); lvl >= len(spaceText) {
			indentText = spaceText
		} else if lvl > 0 {
			indentText = spaceText[:lvl]
		}
		var blockIndicator string
		if tok.Block() {
			blockIndicator = "B"
		} else {
			blockIndicator = "i"
		}

		log.Printf("%03d: %s%s:%#v", idx, indentText, blockIndicator, tok)
	}
}
