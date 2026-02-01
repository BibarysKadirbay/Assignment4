package httpapi

import (
	"net/http"
	"strings"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) Handle(pattern string, h http.Handler) {
	r.mux.Handle(pattern, h)
}

func (r *Router) HandleFunc(pattern string, fn func(http.ResponseWriter, *http.Request)) {
	r.mux.HandleFunc(pattern, fn)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func PathParamAfter(prefix, path string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return "", false
	}
	parts := strings.Split(rest, "/")
	return parts[0], true
}
