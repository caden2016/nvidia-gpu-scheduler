package router

import "net/http"

// Router defines an interface to specify a group of routes to add to the server.
type Router interface {
	// Routes returns the list of routes to add to the server.
	Routes() []Route
	DumpRoutes()
}

// Route defines an individual API route in the server.
type Route interface {
	// Handler returns the http handler.
	Handler() http.HandlerFunc
	// Method returns the http method that the route responds to.
	Method() string
	// Path returns the subpath where the route responds to.
	Path() string
}
