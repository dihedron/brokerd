package server

import (
	"context"

	"github.com/dihedron/brokerd/proto"
)

// KVStoreServer is the server API for KVStore service.
// All implementations must embed UnimplementedKVStoreServer
// for forward compatibility
type KVStoreServer struct {
	proto.UnimplementedKVStoreServer
}

func (s *KVStoreServer) Get(ctx context.Context, pair proto.Pair) (*proto.Pair, error) {
	return nil, nil
}
func (s *KVStoreServer) Set(ctx context.Context, pair *proto.Pair) (*proto.Pair, error) {
	return nil, nil
}
func (s *KVStoreServer) Remove(ctx context.Context, pair *proto.Pair) (*proto.Pair, error) {
	return nil, nil
}
