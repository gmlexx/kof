package server

import (
	"fmt"
	"net/http"
	"time"
)

type Middleware func(Handler) Handler

func Chain(h Handler, middleware ...Middleware) Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func (s *Server) ApplyMiddleware(h Handler) Handler {
	if len(s.middlewares) == 0 {
		return h
	}
	return Chain(h, s.middlewares...)
}

func (s *Server) Use(middleware ...Middleware) {
	s.middlewares = append(s.middlewares, middleware...)
}

func RecoveryMiddleware(next Handler) Handler {
	return func(res *Response, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				res.Fail("Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(res, req)
	}
}

func LoggingMiddleware(next Handler) Handler {
	return func(res *Response, req *http.Request) {
		b := time.Now()
		next(res, req)
		res.Duration = time.Now().Sub(b)
		res.Logger.Info(fmt.Sprintf("\"%s %s\" %d %f", req.Method, req.URL, res.Status, res.Duration.Seconds()))
	}
}
