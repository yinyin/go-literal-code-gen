package literalcodegen

import (
	"log"
)

func logLiteralEntry(entry *LiteralEntry) {
	log.Printf("- %s (depth=%d; mode=%d): trim-space=%v, preserve-new-line=%v, tail-new-line=%v",
		entry.Name,
		entry.LevelDepth,
		entry.TranslationMode,
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
