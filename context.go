package yawf

import (
	"github.com/codegangsta/inject"
	"net/http"
	"reflect"
)

// Context represents a request context. Services can be mapped on the request level from this interface.
type Context interface {
	inject.Injector
	// Next is an optional function that Middleware Handlers can call to yield the until after
	// the other Handlers have been executed. This works really well for any operations that must
	// happen after an http request
	Next()
	// Written returns whether or not the response for this context has been written.
	Written() bool

	Stop()

	IsStopped() bool
}

type context struct {
	inject.Injector
	handlers []Handler
	action   Handler
	rw       ResponseWriter
	index    int
}

func NewContext(handlers []Handler, action Handler, res http.ResponseWriter) Context {
	c := &context{inject.New(), handlers, action, NewResponseWriter(res), -1}
	c.MapTo(c, (*Context)(nil))
	c.MapTo(c.rw, (*http.ResponseWriter)(nil))
	return c
}

func (c *context) Next() {
	c.index += 1
	c.run()
}

func (c *context) Written() bool {
	return c.rw.Written()
}

func (c *context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}
	if c.index == len(c.handlers) {
		return c.action
	}
	panic("invalid index for context handler")
}

func (c *context) Stop() {
	c.index = len(c.handlers) + 1
}

func (c *context) IsStopped() bool {
	return !(c.index <= len(c.handlers))
}

func (c *context) run() {
	for !c.IsStopped() {
		vals, err := c.Invoke(c.handler())
		if err != nil {
			panic(err)
		}

		ev := c.Get(reflect.TypeOf(MiddlewareReturnHandler(nil)))
		handleReturn := ev.Interface().(MiddlewareReturnHandler)
		handleReturn(c, vals)
		c.index += 1

		if c.Written() {
			return
		}
		if c.IsStopped() {
			return
		}
	}
}
