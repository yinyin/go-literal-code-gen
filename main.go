package main

import (
	"log"

	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

func main() {
	inputFilePath, outputFilePath, externalFilter, err := parseCommandParam()
	if nil != err {
		log.Fatalf("ERR: cannot have required parameters: %v", err)
		return
	}
	log.Printf("Input: %v", inputFilePath)
	log.Printf("Output: %v", outputFilePath)
	log.Printf("External Filter: %v", externalFilter)
	code, err := literalcodegen.ParseMarkdown(inputFilePath)
	if nil != err {
		log.Fatalf("ERR: parsing Markdown input failed: %v", err)
		return
	}
	log.Printf("** Loaded input.")
	literalcodegen.LogLiteralCode(code)
	log.Printf("** Going to generate code.")
	err = literalcodegen.GenerateGoCodeFile(outputFilePath, code, externalFilter)
	if nil != err {
		log.Fatalf("ERR: failed on generating output code: %v", err)
		return
	}
	log.Printf("** Completed.")
}
