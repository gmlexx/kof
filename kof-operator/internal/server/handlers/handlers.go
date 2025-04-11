package handlers

import (
	"fmt"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"net/http"
)

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	fmt.Fprintln(res.Writer, "404 - Page not found")
}
