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
	replaceRule *ReplaceRule
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
	return nil, fmt.Errorf("unexpected markdown node (L1-H): %T %#v", token, token)
}

func (w *markdownParseSpace) checkHeading(token *markdown.HeadingOpen) (nextCallable markdownParseCallable, err error) {
	if token.HLevel == 1 {
		return w.stateHeading1, nil
	}
	return nil, nil
}

func (w *markdownParseSpace) stateReplaceRuleZero(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	node, ok := token.(*markdown.CodeInline)
	if !ok {
		log.Printf("- skipped: replace-rule (L2-R-0): %T, %v", token, token)
		return
	}
	txt := node.Content
	if nil == w.replaceRule.RegexTrap {
		err = w.replaceRule.setRegexTrap(txt)
	} else if -1 == w.replaceRule.GroupIndex {
		err = w.replaceRule.setGroupIndex(txt)
	} else {
		w.replaceRule.setReplacementText(txt)
	}
	return
}

func (w *markdownParseSpace) stateReplaceRule(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	switch token.(type) {
	case *markdown.Inline:
		node := token.(*markdown.Inline)
		w.feedTokens(w.stateReplaceRuleZero, node.Children)
	case *markdown.BulletListClose:
		if nil != w.replaceRule.RegexTrap {
			w.currentNode.appendReplaceRule(w.replaceRule)
		}
		w.replaceRule = nil
		return w.stateOptionItem, nil
	default:
		log.Printf("- skipped: replace-rule (L2-R): %T, %v", token, token)
	}
	return
}

func (w *markdownParseSpace) stateOptionItemConst(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	node, ok := token.(*markdown.CodeInline)
	if !ok {
		log.Printf("- skipped: option (L1-0-const): %T, %v", token, token)
		return
	}
	w.currentNode.Name = node.Content
	return
}

func (w *markdownParseSpace) stateOptionItemBuilder(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	node, ok := token.(*markdown.CodeInline)
	if !ok {
		log.Printf("- skipped: option (L1-0-builder): %T, %v", token, token)
		return
	}
	txt := node.Content
	if "" == w.currentNode.Name {
		w.currentNode.Name = txt
	} else {
		w.currentNode.Parameters = append(w.currentNode.Parameters, txt)
	}
	return
}

func (w *markdownParseSpace) stateOptionItemZero(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	node, ok := token.(*markdown.CodeInline)
	if !ok {
		log.Printf("- skipped: option (L1-0): %T, %v", token, token)
		return
	}
	switch node.Content {
	case "const":
		w.currentNode.TranslationMode = TranslateAsConst
		nextCallable = w.stateOptionItemConst
	case "builder":
		w.currentNode.TranslationMode = TranslateAsBuilder
		nextCallable = w.stateOptionItemBuilder
	case "replace":
		w.replaceRule = newReplaceRule()
		// return w.stateOptionItemReplace, nil
	case "strip-spaces":
		w.currentNode.TrimSpace = true
	case "preserve-new-line":
		w.currentNode.PreserveNewLine = true
	case "tail-new-line":
		w.currentNode.TailNewLine = true
	default:
		log.Printf("** unknown option command (L1-0): %v", node.Content)
	}
	return
}

func (w *markdownParseSpace) stateOptionItem(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	switch token.(type) {
	case *markdown.Inline:
		node := token.(*markdown.Inline)
		w.feedTokens(w.stateOptionItemZero, node.Children)
	case *markdown.ListItemClose:
		return w.stateZero, nil
	case *markdown.BulletListOpen:
		return w.stateReplaceRule, nil
	default:
		log.Printf("- skipped option (L1): %T, %#v", token, token)
	}
	return
}

func (w *markdownParseSpace) stateZero(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	switch token.(type) {
	case *markdown.HeadingOpen:
		return w.checkHeading(token.(*markdown.HeadingOpen))
	case *markdown.ListItemOpen:
		return w.stateOptionItem, nil
	case *markdown.Fence:
		fenceToken := token.(*markdown.Fence)
		w.currentNode.AppendContent(fenceToken.Content)
	default:
		log.Printf("- skipped: markdown (L0): %T, %#v", token, token)
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
func ParseMarkdown(filePath string) (code *LiteralCode, err error) {
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
	return &work.result, nil
}
