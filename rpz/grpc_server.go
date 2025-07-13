package rpz

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/dnswlt/hackz/rpz/rpzpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	rpzpb.UnimplementedItemServiceServer

	config Config
	mu     sync.RWMutex
	items  map[string]*rpzpb.Item
}

func NewGRPCServer(config Config) *GRPCServer {
	return &GRPCServer{
		config: config,
		items:  make(map[string]*rpzpb.Item),
	}
}

func (s *GRPCServer) CreateItem(ctx context.Context, req *rpzpb.CreateItemRequest) (*rpzpb.Item, error) {
	item := &rpzpb.Item{
		Id:        req.GetId(),
		Name:      req.GetName(),
		Timestamp: timestamppb.New(time.Now()),
	}

	s.mu.Lock()
	s.items[item.Id] = item
	s.mu.Unlock()

	return item, nil
}

func (s *GRPCServer) GetItem(ctx context.Context, req *rpzpb.GetItemRequest) (*rpzpb.Item, error) {
	s.mu.RLock()
	item, ok := s.items[req.GetId()]
	s.mu.RUnlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "item not found")
	}

	return item, nil
}

func (s *GRPCServer) Serve() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var grpcServer *grpc.Server

	if !s.config.Insecure {
		// TLS
		creds, err := credentials.NewServerTLSFromFile(s.config.CertFile, s.config.KeyFile)
		if err != nil {
			log.Fatalf("failed to load TLS credentials: %v", err)
		}

		grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		// No TLS, insecure!
		grpcServer = grpc.NewServer()
	}

	rpzpb.RegisterItemServiceServer(grpcServer, s)

	log.Println("gRPC server listening on :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}
