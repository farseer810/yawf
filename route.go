package yawf

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
)

// Route is an interface representing a Route in Yawf's routing layer.
type Route interface {
	// URLWith returns a rendering of the Route's url with the given string params.
	URLWith([]string) string
	// SetName sets a name for the route.
	SetName(string)
	// Name returns the name of the route.
	Name() string
	// Pattern returns the pattern of the route.
	Pattern() string
	// Method returns the method of the route.
	Method() string
}

type route struct {
	method   string
	regex    *regexp.Regexp
	handlers []Handler
	pattern  string
	name     string
}

var routeReg1 = regexp.MustCompile(`:[^/#?()\.\\]+`)
var routeReg2 = regexp.MustCompile(`\*\*`)

func newRoute(method string, pattern string, handlers []Handler) *route {
	route := route{method, nil, handlers, pattern, ""}
	pattern = routeReg1.ReplaceAllStringFunc(pattern, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[^/#?]+)`, m[1:])
	})
	var index int
	pattern = routeReg2.ReplaceAllStringFunc(pattern, func(m string) string {
		index++
		return fmt.Sprintf(`(?P<_%d>[^#?]*)`, index)
	})
	pattern += `\/?`
	route.regex = regexp.MustCompile(pattern)
	return &route
}

type RouteMatch int

const (
	NoMatch RouteMatch = iota
	StarMatch
	OverloadMatch
	ExactMatch
)

//Higher number = better match
func (r RouteMatch) BetterThan(o RouteMatch) bool {
	return r > o
}

func (r route) MatchMethod(method string) RouteMatch {
	switch {
	case method == r.method:
		return ExactMatch
	case method == "HEAD" && r.method == "GET":
		return OverloadMatch
	case r.method == "*":
		return StarMatch
	default:
		return NoMatch
	}
}

func (r route) Match(method string, path string) (RouteMatch, map[string]string) {
	// add Any method matching support
	match := r.MatchMethod(method)
	if match == NoMatch {
		return match, nil
	}

	matches := r.regex.FindStringSubmatch(path)
	if len(matches) > 0 && matches[0] == path {
		params := make(map[string]string)
		for i, name := range r.regex.SubexpNames() {
			if len(name) > 0 {
				params[name] = matches[i]
			}
		}
		return match, params
	}
	return NoMatch, nil
}

func (r *route) Validate() {
	for _, handler := range r.handlers {
		ValidateHandler(handler)
	}
}

func (r *route) Handle(c Context, res http.ResponseWriter) {
	context := &routeContext{c, 0, r.handlers}
	c.MapTo(context, (*Context)(nil))
	c.MapTo(r, (*Route)(nil))
	context.run()
}

var urlReg = regexp.MustCompile(`:[^/#?()\.\\]+|\(\?P<[a-zA-Z0-9]+>.*\)`)

// URLWith returns the url pattern replacing the parameters for its values
func (r *route) URLWith(args []string) string {
	if len(args) > 0 {
		argCount := len(args)
		i := 0
		url := urlReg.ReplaceAllStringFunc(r.pattern, func(m string) string {
			var val interface{}
			if i < argCount {
				val = args[i]
			} else {
				val = m
			}
			i += 1
			return fmt.Sprintf(`%v`, val)
		})

		return url
	}
	return r.pattern
}

func (r *route) SetName(name string) {
	r.name = name
}

func (r *route) Name() string {
	return r.name
}

func (r *route) Pattern() string {
	return r.pattern
}

func (r *route) Method() string {
	return r.method
}

type routeContext struct {
	Context
	index    int
	handlers []Handler
}

func (r *routeContext) Next() {
	r.index += 1
	r.run()
}

func (r *routeContext) run() {
	for r.index < len(r.handlers) {
		handler := r.handlers[r.index]
		vals, err := r.Invoke(handler)
		if err != nil {
			panic(err)
		}
		r.index += 1

		// if the handler returned something, write it to the http response
		if len(vals) > 0 {
			ev := r.Get(reflect.TypeOf(RouterReturnHandler(nil)))
			handleReturn := ev.Interface().(RouterReturnHandler)
			handleReturn(r, vals)
			return
		}

		if r.Written() {
			return
		}
	}
}
