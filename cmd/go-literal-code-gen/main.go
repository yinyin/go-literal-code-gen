package main

import (
	"log"

	literalcodegen "github.com/yinyin/go-literal-code-generator"
)

func main() {
	inputFilePath, outputFilePath, err := parseCommandParam()
	if nil != err {
		log.Fatalf("ERR: cannot have required parameters: %v", err)
		return
	}
	log.Printf("Input: %v", inputFilePath)
	log.Printf("Output: %v", outputFilePath)
	code, err := literalcodegen.ParseMarkdown(inputFilePath)
	if nil != err {
		log.Fatalf("ERR: parsing Markdown input failed: %v", err)
		return
	}
	literalcodegen.LogLiteralCode(code)
}
