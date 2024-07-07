package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RouteHandler = func (*Request, *Response)

type Route struct {
	path string
	handler RouteHandler
}

type Server struct {
	routes map[string][]Route
	mu sync.RWMutex
	server *http.Server
	workerPool chan struct{}
}

func NewServer() *Server {
	maxWorkers := runtime.NumCPU() * 100
	return &Server{
		routes: make(map[string][]Route),
		workerPool : make(chan struct{}, maxWorkers),
	}
}

func (s *Server) Listen (port uint16) error  {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	s.server = &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 15 * time.Second,
	}
	return s.server.ListenAndServe()
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	s.workerPool <- struct{}{}
	defer func() {
		<-s.workerPool
	}()

	s.mu.RLock()
	routes,ok := s.routes[r.Method]
	s.mu.RUnlock()

	if !ok {
		http.NotFound(w,r)
		return
	}

	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")

	for _,route := range routes {
		routeParts := strings.Split(strings.Trim(route.path, "/"), "/")
		if len(routeParts) != len(pathParts) {
			// route matching algorithm
			continue
		}

		params := make(map[string]string)
		match := true

		for i, part := range routeParts {
			if strings.HasPrefix(part, ":") {
				params[part[1:]] = pathParts[i]
			} else if part != pathParts[i] {
				match = false
				break
			}
		}
		if match {
			req := &Request{
				httpRequest: r,
				params:      params,
			}
			res := NewResponse(w)
			route.handler(req, res)
			return
		}
	}

	http.NotFound(w, r)
}


func (s *Server) addRoute(method, path string, handler RouteHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.routes[method] = append(s.routes[method], Route{path: path, handler: handler})

}

func (s *Server) Get(route string, handler RouteHandler ) {
	s.addRoute(http.MethodGet, route, handler)
}

func (s *Server) Post(route string, handler RouteHandler) {
	s.addRoute(http.MethodPost, route, handler)
}

func (s *Server) Put(route string, handler RouteHandler) {
	s.addRoute(http.MethodPut, route, handler)
}

func (s *Server) Delete(route string, handler RouteHandler) {
	s.addRoute(http.MethodDelete, route, handler)
}

func (s *Server) Any(route string, handler RouteHandler ) {
	s.Get(route, handler)
	s.Post(route, handler)
	s.Put(route, handler)
	s.Delete(route, handler)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}



type Request struct {
	httpRequest *http.Request
	params map[string]string
}

func (r *Request) Headers() map[string]any {
	headers := make(map[string]any)
	for key, value := range r.httpRequest.Header {
		headers[key] = value
	}
	return headers
}

func (r *Request) Query() map[string]any {
	queries := make(map[string]any)
	for key, value := range r.httpRequest.URL.Query() { // return map[string][]string
		queries[key] = value
	}
	return queries
}

func (r *Request) PathParam(param string) string {
	return r.params[param]
}

// Adjusting the Function definition
func Body[T any](r *Request) *T {
	var result T
	err := json.NewDecoder(r.httpRequest.Body).Decode(&result)
	if err != nil {
		return nil
	}
	return &result
}



type Response struct {
	writer http.ResponseWriter
	headerWritten bool
	status int
	headers map[string]string
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{
		writer: w,
		status: http.StatusOK,
		headers: make(map[string]string),
	}
}

func (r *Response) Header(header, value string) *Response {
	if !r.headerWritten {
		r.headers[header] = value
	}
	return r
}

func (r *Response) Status(HTTPStatus string) *Response {
	statusInt, _ := strconv.Atoi(HTTPStatus)
	r.status = statusInt
	return r
}

func (r *Response) writeHeaders() {
	if !r.headerWritten {
		for k, v := range r.headers {
			r.writer.Header().Set(k, v)
		}
		r.writer.WriteHeader(r.status)
		r.headerWritten = true
	}
}

func (r *Response) End() {
	r.writeHeaders()
	if r.headers["Transfer-Encoding"] == "chunked" {
		fmt.Fprintf(r.writer, "0\r\n\r\n")
	}
}

func (r *Response) Json(resp interface{}) error {
	r.Header("Content-Type", "application/json")
	r.writeHeaders()
	return json.NewEncoder(r.writer).Encode(resp)
}

func (r *Response) Write(data []byte) *Response {
	if !r.headerWritten {
		r.Header("Transfer-Encoding", "chunked")
		r.writeHeaders()
	}

	if len(data) == 0 {
		return r
	}

	fmt.Fprintf(r.writer, "%x\r\n", len(data))
	r.writer.Write(data)
	fmt.Fprint(r.writer, "\r\n")
	return r
}