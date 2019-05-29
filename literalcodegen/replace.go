package literalcodegen

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ReplaceTarget points out matching group and replacement code
type ReplaceTarget struct {
	GroupIndex      int
	ReplacementCode string
}

func (target *ReplaceTarget) setGroupIndex(v string) (err error) {
	v = strings.TrimFunc(v, func(r rune) bool {
		return !unicode.IsNumber(r)
	})
	idx, err := strconv.ParseInt(v, 10, 31)
	if nil != err {
		return
	}
	target.GroupIndex = int(idx)
	if target.GroupIndex < 0 {
		target.GroupIndex = 0
	}
	return nil
}

func (target *ReplaceTarget) setReplacementCode(v string) (err error) {
	target.ReplacementCode = v
	return nil
}

// OrderReplaceTarget is a sorting type for ReplaceTarget
type OrderReplaceTarget []*ReplaceTarget

func (a OrderReplaceTarget) Len() int           { return len(a) }
func (a OrderReplaceTarget) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OrderReplaceTarget) Less(i, j int) bool { return a[i].GroupIndex < a[j].GroupIndex }

// ReplaceRule represent literal replacing rule for generating builder function
type ReplaceRule struct {
	RegexTrap *regexp.Regexp
	Targets   []*ReplaceTarget
}

func newReplaceRule() *ReplaceRule {
	return &ReplaceRule{
		RegexTrap: nil,
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

func (rule *ReplaceRule) addTarget() (target *ReplaceTarget) {
	target = &ReplaceTarget{}
	rule.Targets = append(rule.Targets, target)
	return target
}

func (rule *ReplaceRule) sortTarget() {
	sort.Sort(OrderReplaceTarget(rule.Targets))
}

func (rule *ReplaceRule) doReplace(textLine string) (results []*ReplaceResult, err error) {
	aux := rule.RegexTrap.FindStringSubmatchIndex(textLine)
	if nil == aux {
		return
	}
	targetBoundIndex := len(rule.Targets) - 1
	previousSuffixStart := 0
	for targetIndex, target := range rule.Targets {
		indexIdx := target.GroupIndex * 2
		if (indexIdx + 1) >= len(aux) {
			err = fmt.Errorf("[target-%d] given match group index (%d) out of range (%d/2): rule=%#v, %v", targetIndex, target.GroupIndex, len(aux), rule.RegexTrap, textLine)
			return
		}
		replaceStart := aux[indexIdx]
		suffixStart := aux[indexIdx+1]
		result := &ReplaceResult{
			PrefixLiteral: textLine[previousSuffixStart:replaceStart],
			ReplacedCode:  target.ReplacementCode,
		}
		if targetIndex == targetBoundIndex {
			result.SuffixLiteral = textLine[suffixStart:]
		} else {
			previousSuffixStart = suffixStart
		}
		results = append(results, result)
	}
	return results, nil
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

func (r *ReplaceResult) runReplaceWith(rule *ReplaceRule) (prefixResults []*ReplaceResult, replacedResult *ReplaceResult, suffixResults []*ReplaceResult, err error) {
	replacedResult = &ReplaceResult{
		PrefixLiteral: r.PrefixLiteral,
		ReplacedCode:  r.ReplacedCode,
		SuffixLiteral: r.SuffixLiteral,
	}
	if "" != r.PrefixLiteral {
		if prefixResults, err = rule.doReplace(r.PrefixLiteral); nil != err {
			return
		} else if len(prefixResults) > 0 {
			replacedResult.PrefixLiteral = ""
		}
	}
	if "" != r.SuffixLiteral {
		if suffixResults, err = rule.doReplace(r.SuffixLiteral); nil != err {
			return
		} else if len(suffixResults) > 0 {
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
			prefixResults, replacedResult, suffixResults, err := aux.runReplaceWith(rule)
			if nil != err {
				return nil, err
			}
			if len(prefixResults) > 0 {
				result = append(result, prefixResults...)
			}
			if nil != replacedResult {
				result = append(result, replacedResult)
			}
			if len(suffixResults) > 0 {
				result = append(result, suffixResults...)
			}
		}
	}
	if (1 == len(result)) && result[0].isSimpleLiteral() {
		return nil, nil
	}
	return result, nil
}
