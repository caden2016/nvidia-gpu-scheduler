package router

import (
	"net/http"
)

// RouteWrapper wraps a route with extra functionality.
// It is passed in when creating a new route.
type RouteWrapper func(r Route) Route

// localRoute defines an individual API route to connect
// with the server. It implements Route.
type localRoute struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// Handler returns the APIFunc to let the server wrap it in middlewares.
func (l localRoute) Handler() http.HandlerFunc {
	return l.handler
}

// Method returns the http method that the route responds to.
func (l localRoute) Method() string {
	return l.method
}

// Path returns the subpath where the route responds to.
func (l localRoute) Path() string {
	return l.path
}

// NewRoute initializes a new local route for the router.
func NewRoute(method, path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	var r Route = localRoute{method, path, handler}
	for _, o := range opts {
		r = o(r)
	}
	return r
}

// NewGetRoute initializes a new route with the http method GET.
func NewGetRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodGet, path, handler, opts...)
}

// NewPostRoute initializes a new route with the http method POST.
func NewPostRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodPost, path, handler, opts...)
}

// NewPutRoute initializes a new route with the http method PUT.
func NewPutRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodPut, path, handler, opts...)
}

// NewDeleteRoute initializes a new route with the http method DELETE.
func NewDeleteRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodDelete, path, handler, opts...)
}

// NewOptionsRoute initializes a new route with the http method OPTIONS.
func NewOptionsRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodOptions, path, handler, opts...)
}

// NewHeadRoute initializes a new route with the http method HEAD.
func NewHeadRoute(path string, handler http.HandlerFunc, opts ...RouteWrapper) Route {
	return NewRoute(http.MethodHead, path, handler, opts...)
}
