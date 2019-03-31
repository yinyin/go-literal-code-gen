package literalcodegen

import (
	"io/ioutil"
	"log"

	"gitlab.com/golang-commonmark/markdown"
)

type markdownParseCallable func(token markdown.Token) (markdownParseCallable, error)

type markdownParseSpace struct {
	result       LiteralCode
	currentNode  *LiteralEntry
	nextCallable markdownParseCallable
}

func newMarkdownParseSpace() (result *markdownParseSpace) {
	result = &markdownParseSpace{}
	result.nextCallable = result.stateZero
	return
}

func (w *markdownParseSpace) stateZero(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	log.Printf("> %T: %#v", token, token)
	return nil, nil
}

func (w *markdownParseSpace) feedToken(token markdown.Token) (err error) {
	nextCallable, err := w.nextCallable(token)
	if nil != err {
		return
	}
	if nil != nextCallable {
		w.nextCallable = nextCallable
	}
	return nil
}

// ParseMarkdown parse input file as literal definition in markdown form.
func ParseMarkdown(filePath string) (err error) {
	buf, err := ioutil.ReadFile(filePath)
	if nil != err {
		return
	}
	md := markdown.New()
	tokens := md.Parse(buf)
	work := newMarkdownParseSpace()
	for _, tok := range tokens {
		if err = work.feedToken(tok); nil != err {
			return err
		}
	}
	return nil
}
