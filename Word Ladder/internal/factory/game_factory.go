package factory

type Game struct {
	StartWord   string
	TargetWord  string
	CurrentWord string
	History     map[string]bool // To keep track of the words that have been used in the game
}

type WordPair struct {
	Start  string
	Target string
}

type GameFactory struct{}

// NewGameFactory constructs a factory with default word pairs
func NewGameFactory() *GameFactory {
	return &GameFactory{}
}

func (f *GameFactory) CreateGameWithWords(start, target string) *Game {
	return &Game{
		StartWord:   start,
		TargetWord:  target,
		CurrentWord: start,
		History: map[string]bool{
			start: true,
		},
	}
}
