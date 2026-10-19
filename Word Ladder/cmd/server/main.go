package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"word-ladder/internal/factory"
	"word-ladder/internal/validation"
	pb "word-ladder/proto"
)

type gameServer struct {
	pb.UnimplementedGameServiceServer
	factory    *factory.GameFactory
	validator  validation.Validator
	dictClient pb.DictionaryServiceClient
	mu         sync.Mutex
	active     *factory.Game
}

// newGameServer initializes a new gameServer with the provided dictionary client
func newGameServer(dictClient pb.DictionaryServiceClient) *gameServer {
	return &gameServer{
		factory:    factory.NewGameFactory(),
		validator:  validation.BuildValidationChain(),
		dictClient: dictClient,
	}
}

// StartGame handles gRPC requests to initialize a new game with a random start and target word of the specified length
func (s *gameServer) StartGame(ctx context.Context, req *pb.StartGameRequest) (*pb.StartGameResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	length := req.GetWordLength()
	if length <= 0 {
		length = 4
	}
	if length < 3 || length > 8 {
		return nil, status.Errorf(codes.InvalidArgument, "word length must be between 3 and 8")
	}

	pairResp, err := s.dictClient.GetRandomPair(ctx, &pb.RandomPairRequest{WordLength: length})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch word pair of length %d: %w", length, err)
	}
	if !pairResp.GetSuccess() {
		return nil, status.Errorf(codes.NotFound, "dictionary has no word pair of length %d", length)
	}

	s.active = s.factory.CreateGameWithWords(pairResp.GetStartWord(), pairResp.GetTargetWord())
	log.Printf("New game initialized (%d letters): %s -> %s", length, s.active.StartWord, s.active.TargetWord)

	return &pb.StartGameResponse{
		StartWord:  s.active.StartWord,
		TargetWord: s.active.TargetWord,
		Message:    fmt.Sprintf("Game initialized with %d-letter words!", length),
	}, nil
}

// SubmitWord handles gRPC requests to validate a user's word guess and update the game state accordingly
func (s *gameServer) SubmitWord(ctx context.Context, req *pb.SubmitWordRequest) (*pb.SubmitWordResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active == nil {
		return &pb.SubmitWordResponse{
			IsValid: false,
			Message: "No active game. Please start a new game first.",
		}, nil
	}

	word := strings.ToUpper(strings.TrimSpace(req.GetWord()))

	// Run input through the Chain of Responsibility
	isValid, msg := s.validator.Validate(ctx, word, s.active, s.dictClient)
	if !isValid {
		return &pb.SubmitWordResponse{
			IsValid:     false,
			Message:     msg,
			CurrentWord: s.active.CurrentWord,
			TargetWord:  s.active.TargetWord,
			IsComplete:  false,
		}, nil
	}

	// Update game state on valid move
	s.active.CurrentWord = word
	s.active.History[word] = true

	isComplete := (s.active.CurrentWord == s.active.TargetWord)
	if isComplete {
		msg = fmt.Sprintf("Congratulations! You transformed %s to %s!", s.active.StartWord, s.active.TargetWord)
	}

	return &pb.SubmitWordResponse{
		IsValid:     true,
		Message:     msg,
		CurrentWord: s.active.CurrentWord,
		TargetWord:  s.active.TargetWord,
		IsComplete:  isComplete,
	}, nil
}

func main() {
	// Connect Game Server as a client to the Dictionary Service (:50052)
	dictConn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Dictionary Service: %v", err)
	}
	defer dictConn.Close()

	dictClient := pb.NewDictionaryServiceClient(dictConn)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGameServiceServer(grpcServer, newGameServer(dictClient))

	fmt.Println("Game Server listening on localhost:50051...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
