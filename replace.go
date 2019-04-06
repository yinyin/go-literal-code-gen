package literalcodegen

import (
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
	return nil
}

func (rule *ReplaceRule) setReplacementCode(v string) (err error) {
	rule.ReplacementCode = v
	return nil
}
