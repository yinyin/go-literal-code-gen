package literalcodegen

import (
	"fmt"
	"io/ioutil"
	"log"

	"gitlab.com/golang-commonmark/markdown"
)

type markdownParseCallable func(token markdown.Token) (markdownParseCallable, error)

type markdownParseSpace struct {
	result      LiteralCode
	currentNode *LiteralEntry
}

func newMarkdownParseSpace() (result *markdownParseSpace) {
	result = &markdownParseSpace{}
	return
}

func (w *markdownParseSpace) stateHeading1(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	if textToken, ok := token.(*markdown.Inline); ok {
		if textToken.Content == "Heading Code" {
			w.currentNode = w.result.NewHeadingCode()
			log.Printf("having heading code node")
		} else {
			w.currentNode = w.result.NewLiteralConstant()
			log.Printf("having literal constant node")
		}
		return w.stateZero, nil
	}
	return nil, fmt.Errorf("unexpected markdown node: %T %#v", token, token)
}

func (w *markdownParseSpace) checkHeading(token *markdown.HeadingOpen) (nextCallable markdownParseCallable, err error) {
	if token.HLevel == 1 {
		return w.stateHeading1, nil
	}
	return nil, nil
}

func (w *markdownParseSpace) stateZero(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	log.Printf("> %T: %#v", token, token)
	switch t := token.(type) {
	case *markdown.HeadingOpen:
		return w.checkHeading(token.(*markdown.HeadingOpen))
	case *markdown.BulletListOpen:

	default:
		log.Printf("skipped markdown node: %v, %v", t, token)
	}
	return nil, nil
}

func (w *markdownParseSpace) feedTokens(startCallable markdownParseCallable, tokens []markdown.Token) (err error) {
	currentCallable := startCallable
	for _, tok := range tokens {
		if nextCallable, err := currentCallable(tok); nil != err {
			return err
		} else if nil != nextCallable {
			currentCallable = nextCallable
		}
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
	if err = work.feedTokens(work.stateZero, tokens); nil != err {
		return
	}
	return nil
}
