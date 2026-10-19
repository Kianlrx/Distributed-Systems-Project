WORD LADDER - PROJECT OVERVIEW
==============================

ABOUT THE PROJECT
-----------------
This project is a terminal-based word ladder game built as a small
distributed system in Go. The client communicates with the game server over
gRPC. The game server manages the current game and asks a separate dictionary
service to check whether a submitted word is English.

To play, the user selects a word length from 3 to 8 letters (the default is
4). The game provides a start word and a target word of that length. The
player enters words one at a time, changing exactly one letter on each move.
Each guess must also be a valid English word and must not have been used
earlier in that game. The client displays the current word and counts valid
moves until the target is reached.

HOW THE SERVICES WORK TOGETHER
------------------------------
1. The client prompts for a word length and calls StartGame on the game
   server, which listens on port 50051.
2. The game server requests a random pair of words of that length from the
   dictionary service on port 50052.
3. The client submits each guess to the game server. The server checks the
   move rules and calls the dictionary service to validate the word.
4. The game server returns the validation result and updated game state to
   the client. When the target is reached, the client displays the number
   of valid moves.

THE THREE DESIGN PATTERNS
-------------------------

1. SINGLETON - DICTIONARY STORE
   The dictionary package exposes GetInstance, which loads the word data into
   an in-memory Store shared within the dictionary service process. A
   sync.Once guard ensures initialization is performed at most once.

   Rationale: Loading large JSON word lists for every request would be slow
   and wasteful. Keeping one in-memory store makes lookups fast and avoids
   repeated file reads and parsing.

2. FACTORY - GAME CREATION
   GameFactory.CreateGameWithWords constructs a Game from the start and target
   words. It also initializes the current word to the start word and creates
   the history map with the start word already recorded.

   Rationale: Game initialization belongs in one place. The factory ensures
   every new game begins with the same required state and keeps that setup
   separate from the game server's request-handling code.

3. CHAIN OF RESPONSIBILITY - MOVE VALIDATION
   BuildValidationChain links validators in order:
   word length, one-letter difference, word history, and dictionary lookup.
   Each validator rejects a guess that fails its rule; otherwise it passes
   the guess to the next validator.

   Rationale: Each rule has a focused implementation, and the sequence is
   easy to understand and maintain. Rules can be changed or extended without
   placing every check in one large validation function.

WHY THERE ARE TWO JSON WORD FILES
---------------------------------
The project uses two word lists for different purposes:

* words_dictionary.json is the large validation dictionary. The dictionary
  service uses it to decide whether submitted guesses are English words.
* top_10000_english_words_usa.json is a smaller list of common English words.
  The store uses this list when choosing start and target words for the
  requested word length.

The full dictionary is useful for accepting a broad range of valid
intermediate guesses, but choosing endpoints from it directly can produce
unfamiliar words and make the game less approachable. The common-word list
was added so the endpoints are more recognizable while still allowing the
user to choose their length. The store considers up to the first 2,500
entries from that list and keeps at most 200 words for each length.

RUNNING THE PROJECT
-------------------
Run these commands from the project root so the dictionary service can find
both JSON files using their relative paths. Start each service in its own
terminal:

    go run ./cmd/dictionary
    go run ./cmd/server
    go run ./cmd/client

The dictionary service should be started before the game server, and both
servers should be running before starting the client.

CURRENT LIMITATIONS
-------------------
The game server currently stores a single active game, so multiple clients
would share and affect the same game state. Also, start and target words are
chosen as distinct common words of the requested length; pair selection does
not currently check that a complete sequence of legal moves exists between
them.