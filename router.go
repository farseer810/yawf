package yawf

import (
	"net/http"
	"strconv"
)

// Params is a map of name/value pairs for named routes. An instance of yawf.Params is available to be injected into any route handler.
type PathParams map[string]string

// Routes is a helper service for Yawf's routing layer.
type Routes interface {
	// URLFor returns a rendered URL for the given route. Optional params can be passed to fulfill named parameters in the route.
	URLFor(name string, params ...interface{}) string
	// MethodsFor returns an array of methods available for the path
	MethodsFor(path string) []string
	// All returns an array with all the routes in the router.
	All() []Route
}

// Router is Yawf's de-facto routing interface. Supports HTTP verbs, stacked handlers, and dependency injection.
type Router interface {
	Routes

	// Group adds a group where related routes can be added.
	Group(string, func(Router), ...Handler)
	// Get adds a route for a HTTP GET request to the specified matching pattern.
	Get(string, ...Handler) Route
	// Patch adds a route for a HTTP PATCH request to the specified matching pattern.
	Patch(string, ...Handler) Route
	// Post adds a route for a HTTP POST request to the specified matching pattern.
	Post(string, ...Handler) Route
	// Put adds a route for a HTTP PUT request to the specified matching pattern.
	Put(string, ...Handler) Route
	// Delete adds a route for a HTTP DELETE request to the specified matching pattern.
	Delete(string, ...Handler) Route
	// Options adds a route for a HTTP OPTIONS request to the specified matching pattern.
	Options(string, ...Handler) Route
	// Head adds a route for a HTTP HEAD request to the specified matching pattern.
	Head(string, ...Handler) Route
	// Any adds a route for any HTTP method request to the specified matching pattern.
	Any(string, ...Handler) Route
	// AddRoute adds a route for a given HTTP method request to the specified matching pattern.
	AddRoute(string, string, ...Handler) Route

	// NotFound sets the handlers that are called when a no route matches a request. Throws a basic 404 by default.
	NotFound(...Handler)

	// Handle is the entry point for routing. This is used as a yawf.Handler
	Handle(http.ResponseWriter, *http.Request, Context)
}

type group struct {
	pattern  string
	handlers []Handler
}

type router struct {
	routes    []*route
	notFounds []Handler
	groups    []group
}

func NewRouter() Router {
	return &router{notFounds: []Handler{http.NotFound}, groups: make([]group, 0)}
}

func (r *router) addRoute(method string, pattern string, handlers []Handler) *route {
	if len(r.groups) > 0 {
		groupPattern := ""
		h := make([]Handler, 0)
		for _, g := range r.groups {
			groupPattern += g.pattern
			h = append(h, g.handlers...)
		}

		pattern = groupPattern + pattern
		h = append(h, handlers...)
		handlers = h
	}

	route := newRoute(method, pattern, handlers)
	route.Validate()
	r.appendRoute(route)
	return route
}

func (r *router) appendRoute(rt *route) {
	r.routes = append(r.routes, rt)
}

func (r *router) getRoutes() []*route {
	return r.routes
}

func (r *router) findRoute(name string) *route {
	for _, route := range r.getRoutes() {
		if route.name == name {
			return route
		}
	}

	return nil
}

// URLFor returns the url for the given route name.
func (r *router) URLFor(name string, params ...interface{}) string {
	route := r.findRoute(name)

	if route == nil {
		panic("route not found")
	}

	var args []string
	for _, param := range params {
		switch v := param.(type) {
		case int:
			args = append(args, strconv.FormatInt(int64(v), 10))
		case string:
			args = append(args, v)
		default:
			if v != nil {
				panic("Arguments passed to URLFor must be integers or strings")
			}
		}
	}

	return route.URLWith(args)
}

func (r *router) All() []Route {
	routes := r.getRoutes()
	var ri = make([]Route, len(routes))

	for i, route := range routes {
		ri[i] = Route(route)
	}

	return ri
}

// MethodsFor returns all methods available for path
func (r *router) MethodsFor(path string) []string {
	methods := []string{}
	for _, route := range r.getRoutes() {
		matches := route.regex.FindStringSubmatch(path)
		if len(matches) > 0 && matches[0] == path && !hasMethod(methods, route.method) {
			methods = append(methods, route.method)
		}
	}
	return methods
}

func hasMethod(methods []string, method string) bool {
	for _, v := range methods {
		if v == method {
			return true
		}
	}
	return false
}

func (r *router) Handle(res http.ResponseWriter, req *http.Request, context Context) {
	bestMatch := NoMatch
	var bestVals map[string]string
	var bestRoute *route
	for _, route := range r.getRoutes() {
		match, vals := route.Match(req.Method, req.URL.Path)
		if match.BetterThan(bestMatch) {
			bestMatch = match
			bestVals = vals
			bestRoute = route
			if match == ExactMatch {
				break
			}
		}
	}
	if bestMatch != NoMatch {
		params := PathParams(bestVals)
		context.Map(params)

		bestRoute.Handle(context, res)
		return
	}

	// no routes exist, 404
	c := &routeContext{context, 0, r.notFounds}
	context.MapTo(c, (*Context)(nil))
	c.run()
}

func (r *router) Group(pattern string, fn func(Router), h ...Handler) {
	r.groups = append(r.groups, group{pattern, h})
	fn(r)
	r.groups = r.groups[:len(r.groups)-1]
}

func (r *router) Get(pattern string, h ...Handler) Route {
	return r.addRoute("GET", pattern, h)
}

func (r *router) Patch(pattern string, h ...Handler) Route {
	return r.addRoute("PATCH", pattern, h)
}

func (r *router) Post(pattern string, h ...Handler) Route {
	return r.addRoute("POST", pattern, h)
}

func (r *router) Put(pattern string, h ...Handler) Route {
	return r.addRoute("PUT", pattern, h)
}

func (r *router) Delete(pattern string, h ...Handler) Route {
	return r.addRoute("DELETE", pattern, h)
}

func (r *router) Options(pattern string, h ...Handler) Route {
	return r.addRoute("OPTIONS", pattern, h)
}

func (r *router) Head(pattern string, h ...Handler) Route {
	return r.addRoute("HEAD", pattern, h)
}

func (r *router) Any(pattern string, h ...Handler) Route {
	return r.addRoute("*", pattern, h)
}

func (r *router) AddRoute(method, pattern string, h ...Handler) Route {
	return r.addRoute(method, pattern, h)
}

func (r *router) NotFound(handler ...Handler) {
	r.notFounds = handler
}
