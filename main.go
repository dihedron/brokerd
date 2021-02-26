package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	httpd "github.com/dihedron/brokerd/http"
	"github.com/dihedron/brokerd/log"
	"github.com/dihedron/brokerd/store"
	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
)

// Command line defaults
const (
	DefaultHTTPAddr = ":11000"
	DefaultRaftAddr = ":12000"
)

// Options are the application startup options.
type Options struct {
	NodeID      string `short:"i" long:"id" description:"The unique ID of the node." required:"yes"`
	HTTPAddress string `short:"h" long:"http" description:"Address to listen on for HTTP connections." default:"127.0.0.1:11000"`
	RaftAddress string `short:"r" long:"raft" description:"Address to listen on for Raft RPC." default:"127.0.0.1:12000"`
	JoinAddress string `short:"j" long:"join" description:"Address of the Raft leader." optional:"yes"`
	RaftDir     string `short:"d" long:"dir" description:"Directory to store the Raft state in." required:"yes"`
}

func main() {
	defer log.L.Sync()

	options := Options{}

	parser := flags.NewParser(&options, flags.Default)
	if _, err := parser.Parse(); err != nil {
		log.L.Error("failure parsing command line", zap.Error(err))
		os.Exit(1)
	}

	log.L.Info("raft state directory", zap.String("path", options.RaftDir))
	os.MkdirAll(options.RaftDir, 0o700)

	// r := cluster.New(

	// 	cluster.WithRaftDirectory(options.RaftDir),
	// 	cluster.WithRaftBindAddress(options.RaftAddress),
	// )

	s := store.New(
		store.WithRaftDirectory(options.RaftDir),
		store.WithRaftBindAddress(options.RaftAddress),
	)
	if err := s.Open(options.JoinAddress == "", options.NodeID); err != nil {
		log.L.Error("failed to open store", zap.Error(err))
	}

	h := httpd.New(options.HTTPAddress, s)
	if err := h.Start(); err != nil {
		log.L.Error("failed to start HTTP service", zap.Error(err))
		os.Exit(1)
	}

	// If join was specified, make the join request.
	if options.JoinAddress != "" {
		if err := join(options.JoinAddress, options.RaftAddress, options.NodeID); err != nil {
			log.L.Error("failed to join node", zap.String("join address", options.JoinAddress), zap.Error(err))
		}
	}

	log.L.Info("application started successfully")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.L.Info("application exiting")
}

func join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		log.L.Error("failure marshalling join request nody to JSON", zap.Error(err))
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		log.L.Error("failure sending join request", zap.String("join address", joinAddr), zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	return nil
}
