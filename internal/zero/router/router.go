/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 *
 * Lightweight HTTP router based on net/http.ServeMux (Go 1.22+).
 * Supports path parameters, middleware, and method routing.
 */

package router

import (
	"net/http"
	"strings"
)

// Router is a lightweight HTTP router
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
	prefix      string
}

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// New creates a new router
func New() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// Use adds middleware to the router
func (r *Router) Use(mw Middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// Handle registers a handler for a pattern
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(r.prefix+pattern, r.wrapMiddleware(handler))
}

// HandleFunc registers a handler function for a pattern
func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) {
	r.Handle(pattern, handler)
}

// GET registers a GET handler
func (r *Router) GET(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("GET "+r.prefix+pattern, r.wrapMiddlewareFunc(handler))
}

// POST registers a POST handler
func (r *Router) POST(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("POST "+r.prefix+pattern, r.wrapMiddlewareFunc(handler))
}

// PUT registers a PUT handler
func (r *Router) PUT(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("PUT "+r.prefix+pattern, r.wrapMiddlewareFunc(handler))
}

// DELETE registers a DELETE handler
func (r *Router) DELETE(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("DELETE "+r.prefix+pattern, r.wrapMiddlewareFunc(handler))
}

// Group creates a sub-router with a path prefix
func (r *Router) Group(prefix string) *Router {
	return &Router{
		mux:         r.mux,
		middlewares: append([]Middleware{}, r.middlewares...),
		prefix:      r.prefix + prefix,
	}
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) wrapMiddleware(handler http.Handler) http.Handler {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	return handler
}

func (r *Router) wrapMiddlewareFunc(handler http.HandlerFunc) http.HandlerFunc {
	h := r.wrapMiddleware(handler)
	return h.ServeHTTP
}

// PathValue extracts a path parameter from the request (Go 1.22+)
func PathValue(r *http.Request, name string) string {
	return r.PathValue(name)
}

// Methods returns middleware that restricts to specific HTTP methods
func Methods(methods ...string) Middleware {
	allowed := make(map[string]bool)
	for _, m := range methods {
		allowed[strings.ToUpper(m)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[r.Method] {
				w.Header().Set("Allow", strings.Join(methods, ", "))
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
