package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"word-ladder/internal/dictionary"
	pb "word-ladder/proto"
)

type dictionaryServer struct {
	pb.UnimplementedDictionaryServiceServer
	store *dictionary.Store
}

// GetRandomPair handles gRPC requests for a random start/target word pair of a given length
func (s *dictionaryServer) GetRandomPair(ctx context.Context, req *pb.RandomPairRequest) (*pb.RandomPairResponse, error) {
	start, target, ok := s.store.GetRandomPair(int(req.GetWordLength()))
	if !ok {
		return &pb.RandomPairResponse{Success: false}, nil
	}

	log.Printf("Selected random pair for length %d: %s -> %s", req.GetWordLength(), start, target)
	return &pb.RandomPairResponse{
		StartWord:  start,
		TargetWord: target,
		Success:    true,
	}, nil
}

// ValidateWord handles gRPC requests asking if a string is a real English word
func (s *dictionaryServer) ValidateWord(ctx context.Context, req *pb.WordRequest) (*pb.WordResponse, error) {
	isValid := s.store.IsValid(req.GetWord())
	log.Printf("Lookup requested for '%s' -> Valid: %t", req.GetWord(), isValid)

	return &pb.WordResponse{
		IsValid: isValid,
	}, nil
}

func main() {
	// Initialize the Singleton store using the JSON dataset
	store, err := dictionary.GetInstance("words_dictionary.json")
	if err != nil {
		log.Fatalf("Failed to initialize dictionary store: %v", err)
	}

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen on port 50052: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDictionaryServiceServer(grpcServer, &dictionaryServer{store: store})

	fmt.Println("Dictionary Service listening on localhost:50052...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
