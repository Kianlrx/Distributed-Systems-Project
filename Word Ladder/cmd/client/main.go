package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "word-ladder/proto"
)

func main() {
	// gRPC connection to Game Server on localhost:50051
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Game Server: %v", err)
	}
	defer conn.Close()

	fmt.Println("\n--------------------------------------------")
	fmt.Println("                WORD LADDER                ")
	fmt.Println("--------------------------------------------")

	client := pb.NewGameServiceClient(conn)

	// User input for word length
	reader := bufio.NewReader(os.Stdin)
	var wordLength int32

	for {
		fmt.Print("\nEnter desired word length (3-6) [Default 4]: ")
		inputLenStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading word length: %v", err)
		}
		inputLenStr = strings.TrimSpace(inputLenStr)
		if inputLenStr == "" {
			wordLength = 4
			break
		}

		val, err := strconv.Atoi(inputLenStr)
		if err == nil && val >= 3 && val <= 6 {
			wordLength = int32(val)
			break
		}

		fmt.Println("\nPlease enter a whole number from 3 to 6.")
	}

	// request new game initialisation
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	startResp, err := client.StartGame(startCtx, &pb.StartGameRequest{WordLength: wordLength})
	startCancel()

	if err != nil {
		log.Fatalf("Error initializing game on server: %v", err)
	}

	fmt.Printf("%s\n", startResp.GetMessage())
	fmt.Printf("Start Word  : %s\n", startResp.GetStartWord())
	fmt.Printf("Target Word : %s\n", startResp.GetTargetWord())
	fmt.Println("Type 'exit' or 'quit' anytime to leave.")
	fmt.Println("\n----------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	currentWord := startResp.GetStartWord()
	moves := 0

	// Input loop for user guesses
	for {
		fmt.Print("\n", currentWord, " --> ", startResp.GetTargetWord())
		fmt.Print("\nEnter next word > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// check for player exit
		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			fmt.Println("Exiting game session. Goodbye!")
			break
		}

		// send guess to game server for validation
		reqCtx, reqCancel := context.WithTimeout(context.Background(), 5*time.Second)
		subResp, err := client.SubmitWord(reqCtx, &pb.SubmitWordRequest{Word: input})
		reqCancel()

		if err != nil {
			fmt.Printf("Error communicating with Game Server: %v\n", err)
			continue
		}
		if subResp.GetCurrentWord() != "" {
			currentWord = subResp.GetCurrentWord()
		}

		// display validation results
		if !subResp.GetIsValid() {
			fmt.Printf("\n✘ %s\n", subResp.GetMessage())
			continue
		}

		moves++

		if subResp.GetIsComplete() {
			fmt.Println("\n----------------------------------------")
			fmt.Printf("   Success! You've reached the target word in %d moves!\n", moves)
			fmt.Println("----------------------------------------")
			break
		}

		fmt.Printf("\n✔ %s\n", subResp.GetMessage())
		fmt.Printf("Current State: %s -> %s\n", subResp.GetCurrentWord(), subResp.GetTargetWord())

		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
	}
}
