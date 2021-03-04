package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/dihedron/brokerd/cluster"
	"github.com/dihedron/brokerd/kvstore"
	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/web/openapi"
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
	router.Use(
		ginzap.Ginzap(log.L, time.RFC3339, true),
		ginzap.RecoveryWithZap(log.L, true),
		func(ctx *gin.Context) {
			// inject global variables into gin Context
			ctx.Set("store", store)
			ctx.Set("cluster", cluster)
		},
	)
	// register Properties API, Cluster API and Store API
	openapi.AddAPIHandlers(router)

	// // Add the routes that do not need instrumentation
	// api := router.Group("/api/v1/")
	// {
	// 	fleet := api.Group("/fleet")
	// 	{
	// 		fleet.GET("/fleet/self/state", func(c *gin.Context) {
	// 			// var state raft.RaftState
	// 			state := cluster.Raft.State()
	// 			c.JSON(http.StatusOK, gin.H{
	// 				"state": state,
	// 			})
	// 		})

	// 		fleet.GET("/fleet/nodes", func(c *gin.Context) {
	// 			// state := cluster.Raft.State()
	// 			// state.
	// 		})
	// 	}

	// 	api.GET("/configuration/:key", func(c *gin.Context) {
	// 		key := c.Param("key")
	// 		value, err := store.Get(key)
	// 		if err != nil {
	// 			log.L.Error("error retrieving value from store", zap.String("key", key), zap.Error(err))
	// 			c.JSON(http.StatusNotFound, gin.H{
	// 				"code":    "not found",
	// 				"message": "item not found",
	// 			})
	// 			return
	// 		}
	// 		c.JSON(http.StatusOK, gin.H{
	// 			key:   key,
	// 			value: value,
	// 		})
	// 	})
	// }

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
