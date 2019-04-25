package literalcodegen

import (
	"os"
)

// ExternalFilter define interface for external code generating filter
type ExternalFilter interface {
	// PreCodeGenerate is invoked before literal code generation
	PreCodeGenerate(entries []*LiteralEntry) (err error)

	// GenerateExternalCode is invoked after literal code generation
	GenerateExternalCode(fp *os.File, entries []*LiteralEntry) (err error)
}

// ExternalFilterList is an ExternalFilter implementation to support multiple filters
type ExternalFilterList struct {
	Filters []ExternalFilter
}

// PreCodeGenerate is invoked before literal code generation
func (l *ExternalFilterList) PreCodeGenerate(entries []*LiteralEntry) (err error) {
	for _, filter := range l.Filters {
		if err = filter.PreCodeGenerate(entries); nil != err {
			return err
		}
	}
	return nil
}

// GenerateExternalCode is invoked after literal code generation
func (l *ExternalFilterList) GenerateExternalCode(fp *os.File, entries []*LiteralEntry) (err error) {
	for _, filter := range l.Filters {
		if err = filter.GenerateExternalCode(fp, entries); nil != err {
			return err
		}
	}
	return nil
}
