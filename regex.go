package rio

import (
	"context"
	"regexp"
	"sync"

	"github.com/hungdv136/rio/internal/log"
)

var defaultRegexCompiler = &regexCompiler{}

type regexCompiler struct {
	exprs map[string]*regexp.Regexp
	l     sync.RWMutex
}

func (c *regexCompiler) compile(ctx context.Context, expr string) (*regexp.Regexp, error) {
	if r := c.getFromCache(expr); r != nil {
		return r, nil
	}

	c.l.Lock()
	defer c.l.Unlock()

	r, err := regexp.Compile(expr)
	if err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	if c.exprs == nil {
		c.exprs = map[string]*regexp.Regexp{}
	}

	c.exprs[expr] = r
	return r, nil
}

func (c *regexCompiler) getFromCache(expr string) *regexp.Regexp {
	c.l.RLock()
	defer c.l.RUnlock()

	if r, ok := c.exprs[expr]; ok {
		return r
	}

	return nil
}
