package validation

import (
	"context"
	"fmt"
	"strings"

	"word-ladder/internal/factory"
	pb "word-ladder/proto"
)

// Validator defines the handler interface for the chain
type Validator interface {
	SetNext(next Validator) Validator
	Validate(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string)
}

// BaseValidator provides the default next-link execution
type BaseValidator struct {
	next Validator
}

func (b *BaseValidator) SetNext(next Validator) Validator {
	b.next = next
	return next
}

func (b *BaseValidator) ValidateNext(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string) {
	if b.next != nil {
		return b.next.Validate(ctx, word, game, dictClient)
	}
	return true, "Valid move!"
}

// Length Check
type LengthValidator struct {
	BaseValidator
}

func (v *LengthValidator) Validate(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string) {
	if len(word) != len(game.CurrentWord) {
		return false, fmt.Sprintf("Word must be exactly %d letters long.", len(game.CurrentWord))
	}
	return v.ValidateNext(ctx, word, game, dictClient)
}

// One-Letter Difference Check
type OneLetterDiffValidator struct {
	BaseValidator
}

func (v *OneLetterDiffValidator) Validate(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string) {
	diff := 0
	for i := 0; i < len(word); i++ {
		if word[i] != game.CurrentWord[i] {
			diff++
		}
	}
	if diff != 1 {
		return false, "Word must change exactly one letter from current word."
	}
	return v.ValidateNext(ctx, word, game, dictClient)
}

// History Check
type HistoryValidator struct {
	BaseValidator
}

func (v *HistoryValidator) Validate(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string) {
	if game.History[strings.ToUpper(word)] {
		return false, "Word has already been used in this game."
	}
	return v.ValidateNext(ctx, word, game, dictClient)
}

// Dictionary Check (RPC to Dictionary Service on :50052)
type DictionaryValidator struct {
	BaseValidator
}

func (v *DictionaryValidator) Validate(ctx context.Context, word string, game *factory.Game, dictClient pb.DictionaryServiceClient) (bool, string) {
	resp, err := dictClient.ValidateWord(ctx, &pb.WordRequest{Word: word})
	if err != nil {
		return false, "Error contacting Dictionary service."
	}
	if !resp.GetIsValid() {
		return false, fmt.Sprintf("'%s' is not a valid English word.", word)
	}
	return v.ValidateNext(ctx, word, game, dictClient)
}

// BuildValidationChain links all validators together in sequence
func BuildValidationChain() Validator {
	length := &LengthValidator{}
	diff := &OneLetterDiffValidator{}
	history := &HistoryValidator{}
	dict := &DictionaryValidator{}

	// Wire chain: Length -> Diff -> History -> Dictionary
	length.SetNext(diff).SetNext(history).SetNext(dict)

	return length
}
