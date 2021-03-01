package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/dihedron/brokerd/cluster"
	"github.com/dihedron/brokerd/kvstore"
	"github.com/dihedron/brokerd/log"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP web server.
type Server struct {
	server   *http.Server
	listener net.Listener
	store    kvstore.KVStore
	cluster  *cluster.Cluster
}

// TODO: consider using https://github.com/Depado/ginprom
// to add Prometheus instrumentation.

// New creates a new WebServer struct and starts the network
// connections listener on the provided address.
func New(address string, store kvstore.KVStore, cluster *cluster.Cluster) (*Server, error) {

	if address == "" {
		log.L.Debug("using default address for HTTP server")
		address = ":8080"
	}
	log.L.Debug("creating HTTP server", zap.String("address", address))

	router := gin.New()
	router.Use(ginzap.Ginzap(log.L, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(log.L, true))
	// Add the routes that do not need instrumentation
	api := router.Group("/api/v1/")
	{
		api.GET("/value/:key", func(c *gin.Context) {
			key := c.Param("key")
			value, err := store.Get(key)
			if err != nil  {
				log.L.Error("error retrieving value from store", zap.Key("key", key), zap.Error(err))
				c.JSON(http.StatusNotFound, gin.H({
					code: "not found",
					message: "item not found",
				}))
				return
			}
			c.JSON(http.StatusOK,  gin.H({
				key: key,
				value: value,
			}))
		})

	}

	// TODO: add more routes here....

	return &Server{
		server: &http.Server{
			Addr:    address,
			Handler: router,
		},
		store:   store,
		cluster: cluster,
	}, nil
}

// Start starts the web server; it is blocking, so it ok to call
// in in a separate goroutine. In order to stop it gracefully,
// use the Stop() function.
func (w *Server) Start() error {
	if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.L.Error("error starting web server", zap.Error(err))
		return err
	}
	return nil
}

// Stop cancels the web server gracefully.
func (w *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.server.Shutdown(ctx); err != nil {
		log.L.Error("error shutting down server", zap.Error(err))
		return err
	}
	log.L.Debug("server exiting")
	return nil
}
