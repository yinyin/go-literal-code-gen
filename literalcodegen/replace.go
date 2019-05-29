package literalcodegen

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ReplaceRule represent literal replacing rule for generating builder function
type ReplaceRule struct {
	RegexTrap       *regexp.Regexp
	GroupIndex      int
	ReplacementCode string
}

func newReplaceRule() *ReplaceRule {
	return &ReplaceRule{
		RegexTrap:       nil,
		GroupIndex:      -1,
		ReplacementCode: "",
	}
}

func (rule *ReplaceRule) setRegexTrap(v string) (err error) {
	v = strings.TrimSpace(v)
	regexRule, err := regexp.Compile(v)
	if nil != err {
		return err
	}
	rule.RegexTrap = regexRule
	return nil
}

func (rule *ReplaceRule) setGroupIndex(v string) (err error) {
	v = strings.TrimFunc(v, func(r rune) bool {
		return !unicode.IsNumber(r)
	})
	idx, err := strconv.ParseInt(v, 10, 31)
	if nil != err {
		return
	}
	rule.GroupIndex = int(idx)
	if rule.GroupIndex < 0 {
		rule.GroupIndex = 0
	}
	return nil
}

func (rule *ReplaceRule) setReplacementCode(v string) (err error) {
	rule.ReplacementCode = v
	return nil
}

func (rule *ReplaceRule) doReplace(textLine string) (result *ReplaceResult, err error) {
	aux := rule.RegexTrap.FindStringSubmatchIndex(textLine)
	if nil == aux {
		return
	}
	indexIdx := rule.GroupIndex * 2
	if (indexIdx + 1) >= len(aux) {
		err = fmt.Errorf("given match group index (%d) out of range (%d/2): rule=%#v, %v", rule.GroupIndex, len(aux), rule.RegexTrap, textLine)
		return
	}
	replaceStart := aux[indexIdx]
	suffixStart := aux[indexIdx+1]
	result = &ReplaceResult{
		PrefixLiteral: textLine[0:replaceStart],
		ReplacedCode:  rule.ReplacementCode,
		SuffixLiteral: textLine[suffixStart:],
	}
	return result, nil
}

// ReplaceResult is thre result of one replace operation.
type ReplaceResult struct {
	PrefixLiteral string
	ReplacedCode  string
	SuffixLiteral string
}

func (r *ReplaceResult) isEmpty() bool {
	if ("" == r.PrefixLiteral) && ("" == r.ReplacedCode) && ("" == r.SuffixLiteral) {
		return true
	}
	return false
}

func (r *ReplaceResult) isSimpleLiteral() bool {
	if ("" != r.PrefixLiteral) && ("" == r.ReplacedCode) && ("" == r.SuffixLiteral) {
		return true
	}
	return false
}

func (r *ReplaceResult) runReplaceWith(rule *ReplaceRule) (prefixResult, replacedResult, suffixResult *ReplaceResult, err error) {
	replacedResult = &ReplaceResult{
		PrefixLiteral: r.PrefixLiteral,
		ReplacedCode:  r.ReplacedCode,
		SuffixLiteral: r.SuffixLiteral,
	}
	if "" != r.PrefixLiteral {
		if prefixResult, err = rule.doReplace(r.PrefixLiteral); nil != err {
			return
		} else if nil != prefixResult {
			replacedResult.PrefixLiteral = ""
		}
	}
	if "" != r.SuffixLiteral {
		if suffixResult, err = rule.doReplace(r.SuffixLiteral); nil != err {
			return
		} else if nil != suffixResult {
			replacedResult.SuffixLiteral = ""
		}
	}
	if replacedResult.isEmpty() {
		replacedResult = nil
	}
	return
}

func doReplace(rules []*ReplaceRule, text string) (result []*ReplaceResult, err error) {
	if nil == rules {
		return nil, nil
	}
	result = []*ReplaceResult{
		{
			PrefixLiteral: text,
			ReplacedCode:  "",
			SuffixLiteral: "",
		}}
	for _, rule := range rules {
		buffer := result
		result = nil
		for _, aux := range buffer {
			prefixResult, replacedResult, suffixResult, err := aux.runReplaceWith(rule)
			if nil != err {
				return nil, err
			}
			if nil != prefixResult {
				result = append(result, prefixResult)
			}
			if nil != replacedResult {
				result = append(result, replacedResult)
			}
			if nil != suffixResult {
				result = append(result, suffixResult)
			}
		}
	}
	if (1 == len(result)) && result[0].isSimpleLiteral() {
		return nil, nil
	}
	return result, nil
}
