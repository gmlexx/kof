package main

import (
	"flag"
	"net/http"
	"os"
	"sync"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	setupLog      = ctrl.Log.WithName("setup")
	shutdownLog   = ctrl.Log.WithName("shutdown")
	httpServerLog = ctrl.Log.WithName("http-server")
)

func main() {

	var httpServerAddr string
	flag.StringVar(&httpServerAddr, "http-server-address", ":9090", "The address the http server endpoint binds to.")

	httpServer := server.NewServer(httpServerAddr, &httpServerLog)
	httpServer.Use(server.RecoveryMiddleware)
	httpServer.Use(server.LoggingMiddleware)
	httpServer.Router.GET("/*", handlers.ReactAppHandler)
	httpServer.Router.GET("/assets/*", handlers.ReactAppHandler)
	httpServer.Router.GET("/api/test", handlers.PrometheusJobsHandler)
	httpServer.Router.NotFound(handlers.NotFoundHandler)

	// setupLog.Info(fmt.Sprintf("Starting http server on %s", httpServerAddr))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Run(); err != nil {
			if err != http.ErrServerClosed {
				// setupLog.Error(err, "Error starting http server")
				os.Exit(1)
			}
		}
	}()

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// if err := httpServer.Shutdown(ctx); err != nil {
	// 	shutdownLog.Error(err, "Http server forced to shutdown")
	// 	os.Exit(1)
	// }
	wg.Wait()
}
