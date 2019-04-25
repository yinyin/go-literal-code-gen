package sqlschema

import (
	"os"

	"github.com/yinyin/go-literal-code-gen/literalcodegen"
)

// CodeGenerateFilter filter and adjust literal entities for generating
// SQL schema code module
type CodeGenerateFilter struct {
}

// NewCodeGenerateFilter create an instance of CodeGenerateFilter
func NewCodeGenerateFilter() (filter *CodeGenerateFilter) {
	return &CodeGenerateFilter{}
}

// PreCodeGenerate is invoked before literal code generation
func (filter *CodeGenerateFilter) PreCodeGenerate(entries []*literalcodegen.LiteralEntry) (err error) {
	for _, entry := range entries {
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (filter *CodeGenerateFilter) GenerateExternalCode(fp *os.File, entries []*literalcodegen.LiteralEntry) (err error) {
	for _, entry := range entries {
	}
	return nil
}
