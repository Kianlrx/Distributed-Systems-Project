// connect to the server
// start the game
// run terminal loop
// render feedback
// handle exit

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.goland.org/grpc/credentials/insecure"

	pb "word-ladder/proto"
)

func main() {
	// Open a connection to the Game Server (on localhost:50051)
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the server: %v", err)
	}
	defer conn.Close() // Closing the connection upon failure

	// Create the gRPC client stub using the generated code from the proto file
	client := pb.NewGameServiceClient(conn)

	// Create a Context with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the game by calling the StartGame RPC method on the server
	response, err:= client.StartGame(ctx, &pb.StartGameRequest{})
	if err != nil {
		log.Fatalf("Failed to start the game: %v", err)
	}
	// Use the response from the server to print the start and target words
	fmt.Printf("Connected to the server. %s\n", response.Message)
	fmt.Printf("Start Word: %s\n", response.StartWord)
	fmt.Printf("Target Word: %s\n", response.TargetWord)
}