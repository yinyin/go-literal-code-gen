package literalcodegen

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"gitlab.com/golang-commonmark/markdown"
)

// TextTrapHeadingCode is the trapping constant string for detecting heading code block
const TextTrapHeadingCode = "Heading Code"

// TextTrapBuilderPrepare is trapping constant for prepare code of builder function.
const TextTrapBuilderPrepare = "- Builder Prepare"

// TextTrapContentCode is optional trapping constant for content code
const TextTrapContentCode = "- Content Code"

// MaxHeadingDepth is the max supported depth level of heading
const MaxHeadingDepth = 6

type markdownParseCallable func(token markdown.Token) (markdownParseCallable, error)

type markdownParseSpace struct {
	result        LiteralCode
	currentNode   *LiteralEntry
	currentChain  [MaxHeadingDepth]*LiteralEntry
	replaceRule   *ReplaceRule
	replaceTarget *ReplaceTarget
}

func newMarkdownParseSpace() (result *markdownParseSpace) {
	result = &markdownParseSpace{}
	return
}

func (w *markdownParseSpace) wipeChainFrom(index int) {
	for idx := index; idx < MaxHeadingDepth; idx++ {
		w.currentChain[idx] = nil
	}
}

func (w *markdownParseSpace) stateHeading1(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	if textToken, ok := token.(*markdown.Inline); ok {
		if textToken.Content == TextTrapHeadingCode {
			w.currentNode = w.result.NewHeadingCode()
			log.Printf("having heading code node")
		} else {
			node := w.result.NewLiteralConstant()
			node.TitleText = textToken.Content
			w.currentNode = node
			w.currentChain[0] = node
			log.Printf("having literal constant node (level=1)")
		}
		return w.stateZero, nil
	}
	return nil, fmt.Errorf("unexpected markdown node (L1-H): %T %#v", token, token)
}

func (w *markdownParseSpace) stateHeading2(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	if textToken, ok := token.(*markdown.Inline); ok {
		switch textToken.Content {
		case TextTrapBuilderPrepare:
			if w.currentNode == nil {
				return nil, fmt.Errorf("expecting base node for builder prepare (L2-H): %T %#v", token, token)
			}
			if w.currentNode.BuilderPrepare != nil {
				log.Printf("WARN: builder prepare already existed (L2-H): [%s] %T %#v", w.currentNode.TitleText, token, token)
			}
			node := w.currentNode.GetBuilderPrepareNode()
			node.TitleText = textToken.Content
			w.currentNode = node
			return w.stateZero, nil
		case TextTrapContentCode:
			if w.currentNode == nil {
				return nil, fmt.Errorf("expecting a working node for builder prepare (L2-H): %T %#v", token, token)
			}
			if w.currentNode.SubWork != NotSubWork {
				w.currentNode = w.currentNode.ParentEntry
				log.Printf("DEBUG: restore !!")
			}
			return w.stateZero, nil
		}
	}
	w.wipeChainFrom(2 - 1)
	if nil == w.currentChain[0] {
		return nil, fmt.Errorf("node with depth should have parent node (L2-H): %#v", token)
	}
	return w.stateHeadingN(token)
}

func (w *markdownParseSpace) stateHeadingN(token markdown.Token) (nextCallable markdownParseCallable, err error) {
	if textToken, ok := token.(*markdown.Inline); ok {
		node := w.result.NewLiteralConstant()
		node.TitleText = textToken.Content
		parentNode := w.currentChain[0]
		for idx := 1; idx < MaxHeadingDepth; idx++ {
			if nil != w.currentChain[idx] {
				parentNode = w.currentChain[idx]
				continue
			}
			node.attachToParent(parentNode)
			w.currentNode = node
			w.currentChain[idx] = node
			log.Printf("having literal constant node (depth=%d)", idx)
			return w.stateZero, nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected markdown node (L-N-H): %T %#v", token, token)
}

func (w *markdownParseSpace) checkHeading(token *markdown.HeadingOpen) (nextCallable markdownParseCallable, err error) {
	if token.HLevel == 1 {
		w.wipeChainFrom(0)
		return w.stateHeading1, nil
	} else if token.HLevel == 2 {
		return w.stateHeading2, nil
	} else if token.HLevel <= MaxHeadingDepth {
		w.wipeChainFrom(token.HLevel - 1)
		if nil == w.currentChain[0] {
			return nil, fmt.Errorf("node with depth should have parent node: %#v", token)
		}
		return w.stateHeadingN, nil
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
		w.replaceTarget = nil
	} else if nil == w.replaceTarget {
		w.replaceTarget = w.replaceRule.addTarget()
		err = w.replaceTarget.setGroupIndex(txt)
	} else {
		w.replaceTarget.setReplacementCode(txt)
		w.replaceTarget = nil
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
			w.replaceRule.sortTarget()
			w.currentNode.appendReplaceRule(w.replaceRule)
		}
		w.replaceRule = nil
		w.replaceTarget = nil
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
	case "noop":
		w.currentNode.TranslationMode = TranslateAsExplicitNoop
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
	case "keep-empty-line":
		w.currentNode.KeepEmptyLine = true
	case "tail-new-line":
		w.currentNode.TailNewLine = true
	case "disable-language-filter":
		w.currentNode.DisableLanguageFilter = true
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
		langType, filterArgs := parseCodeBlockLanguageParams(fenceToken.Params)
		w.currentNode.AppendContent(fenceToken.Content, langType, filterArgs)
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

func parseCodeBlockLanguageParams(params string) (languageType string, filterArgs []string) {
	aux := strings.Split(params, " ")
	for _, arg := range aux {
		if "" == arg {
			continue
		}
		if "" == languageType {
			languageType = arg
		} else {
			filterArgs = append(filterArgs, arg)
		}
	}
	return
}

// ParseMarkdown parse input file as literal definition in markdown form.
func ParseMarkdown(filePath string) (code *LiteralCode, err error) {
	buf, err := ioutil.ReadFile(filePath)
	if nil != err {
		return
	}
	md := markdown.New()
	tokens := md.Parse(buf)
	logMarkdownAST(tokens)
	work := newMarkdownParseSpace()
	if err = work.feedTokens(work.stateZero, tokens); nil != err {
		return
	}
	return &work.result, nil
}
