package literalcodegen

// TranslateAsConst set translation mode in constant
const TranslateAsConst = 1

// LiteralEntry represent one literal entity to generate
type LiteralEntry struct {
	Name            string
	TranslationMode int
	TrimSpace       bool
	TailNewLine     bool
	Content         []string
}

// NewLiteralEntry create a new instance of LiteralEntry and set properties to default values
func NewLiteralEntry() *LiteralEntry {
	return &LiteralEntry{}
}

// LiteralCode represent one literal code module to generate
type LiteralCode struct {
	HeadingCodes     []*LiteralEntry
	LiteralConstants []*LiteralEntry
}

// NewHeadingCode allocate and append one literal entry as heading code node
func (l *LiteralCode) NewHeadingCode() (allocated *LiteralEntry) {
	allocated = NewLiteralEntry()
	l.HeadingCodes = append(l.HeadingCodes, allocated)
	return allocated
}

// NewLiteralConstant allocate and append one literal entry as literal constant node
func (l *LiteralCode) NewLiteralConstant() (allocated *LiteralEntry) {
	allocated = NewLiteralEntry()
	l.LiteralConstants = append(l.LiteralConstants, allocated)
	return allocated
}
