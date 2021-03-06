package completion

import (
	"errors"
	"strings"

	"github.com/elves/elvish/eval"
	"github.com/elves/elvish/eval/vals"
	"github.com/elves/elvish/util"
	"github.com/xiaq/persistent/hashmap"
)

var (
	errIncorrectNumOfResults    = errors.New("matcher must return a bool for each candidate")
	errMatcherMustBeFn          = errors.New("matcher must be a function")
	errMatcherInputMustBeString = errors.New("matcher input must be string")
)

var (
	matchPrefix = eval.NewBuiltinFn(
		"edit:match-prefix", wrapMatcher(strings.HasPrefix))
	matchSubstr = eval.NewBuiltinFn(
		"edit:match-substr", wrapMatcher(strings.Contains))
	matchSubseq = eval.NewBuiltinFn(
		"edit:match-subseq", wrapMatcher(util.HasSubseq))
)

func lookupMatcher(m hashmap.Map, name string) (eval.Callable, bool) {
	key := name
	if !hashmap.HasKey(m, key) {
		// Use fallback matcher
		if !hashmap.HasKey(m, "") {
			return nil, false
		}
		key = ""
	}
	value, _ := m.Index(key)
	matcher, ok := value.(eval.Callable)
	return matcher, ok
}

type matcherOpts struct {
	IgnoreCase bool
	SmartCase  bool
}

func (*matcherOpts) SetDefaultOptions() {}

func wrapMatcher(matcher func(s, p string) bool) interface{} {
	return func(fm *eval.Frame,
		opts matcherOpts, pattern string, inputs eval.Inputs) {

		switch {
		case opts.IgnoreCase && opts.SmartCase:
			throwf("-ignore-case and -smart-case cannot be used together")
		case opts.IgnoreCase:
			innerMatcher := matcher
			matcher = func(s, p string) bool {
				return innerMatcher(strings.ToLower(s), strings.ToLower(p))
			}
		case opts.SmartCase:
			innerMatcher := matcher
			matcher = func(s, p string) bool {
				if p == strings.ToLower(p) {
					// Ignore case is pattern is all lower case.
					return innerMatcher(strings.ToLower(s), p)
				} else {
					return innerMatcher(s, p)
				}
			}
		}

		out := fm.OutputChan()
		inputs(func(v interface{}) {
			s, ok := v.(string)
			if !ok {
				throw(errMatcherInputMustBeString)
			}
			out <- vals.Bool(matcher(s, pattern))
		})
	}
}
