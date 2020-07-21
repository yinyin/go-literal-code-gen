package main

import (
	"errors"
	"flag"
	"path/filepath"

	"github.com/yinyin/go-literal-code-gen/external-filter/sqlschema"
	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

// ErrInputFileRequired indicates input file path is missing.
var ErrInputFileRequired = errors.New("Input file is required")

// ErrOutputFileRequired indicates output file path is missing.
var ErrOutputFileRequired = errors.New("Output file is required")

func parseCommandParam() (inputFilePath, outputFilePath string, genDoNotEdit bool, externalFilter literalcodegen.ExternalFilter, err error) {
	var useSQLSchemaFilter bool
	flag.StringVar(&inputFilePath, "in", "", "path to input file")
	flag.StringVar(&outputFilePath, "out", "", "path to output file")
	flag.BoolVar(&genDoNotEdit, "do-not-edit", false, "generate DO-NOT-EDIT code line")
	flag.BoolVar(&useSQLSchemaFilter, "sqlschema", false, "enable SQL schema filter")
	flag.Parse()
	if "" == inputFilePath {
		err = ErrInputFileRequired
		return
	}
	if inputFilePath, err = filepath.Abs(inputFilePath); nil != err {
		return
	}
	if "" == outputFilePath {
		err = ErrOutputFileRequired
		return
	}
	if outputFilePath, err = filepath.Abs(outputFilePath); nil != err {
		return
	}
	if useSQLSchemaFilter {
		externalFilter = sqlschema.NewCodeGenerateFilter()
	}
	err = nil
	return
}
