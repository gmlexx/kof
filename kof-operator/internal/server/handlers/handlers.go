package handlers

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	static "github.com/k0rdent/kof/kof-operator/webapp/collector"
)

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	fmt.Fprintln(res.Writer, "404 - Page not found")
}

func ReactAppHandler(res *server.Response, req *http.Request) {
	if serveStaticFile(res, req, static.ReactFS) {
		return
	}
	NotFoundHandler(res, req)
}

func PrometheusJobsHandler(res *server.Response, req *http.Request) {
	client, err := k8s.NewClient()
	if err != nil {
		res.Fail(fmt.Sprintf("failed to create client:", err), http.StatusInternalServerError)
		return
	}

	ctx := context.TODO()

	secretList, err := k8s.GetRemoteSecrets(ctx, client)
	if err != nil {
		res.Fail("failed to get secret list", http.StatusInternalServerError)
		return
	}

	kubeconfigs := k8s.GetKubeconfigFromSecretList(secretList)

	responses := make([]byte, 0)
	for _, kubeconfig := range kubeconfigs {
		client, err := k8s.NewKubeClientFromKubeconfig(kubeconfig)
		if err != nil {
			log.Println("failed to create client:", err)
			continue
		}

		podList, err := k8s.GetCollectorPods(ctx, client.Client)
		if err != nil {
			log.Println("failed to list pods:", err)
			continue
		}

		for _, pod := range podList.Items {
			response, err := k8s.Proxy(ctx, client.Clientset, pod, "9090", "/api/v1/targets")
			if err != nil {
				continue
			}

			responses = append(responses, response...)
		}
	}

	res.Writer.Header().Set("Content-Type", "application/json")
	res.SetStatus(http.StatusOK)
	fmt.Fprintln(res.Writer, responses)
}

func serveStaticFile(res *server.Response, req *http.Request, staticFS fs.FS) bool {
	filePath := strings.TrimPrefix(path.Clean(req.URL.Path), "/")
	if filePath == "" {
		filePath = "index.html"
	}

	file, err := staticFS.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		return false
	}

	contentType := getContentType(filePath)
	res.Writer.Header().Set("Content-Type", contentType)

	http.ServeContent(res.Writer, req, filePath, stat.ModTime(), file.(io.ReadSeeker))
	return true
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html"
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "text/plain"
	}
}
