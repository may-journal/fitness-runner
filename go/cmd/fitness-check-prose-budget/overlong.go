package check

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/comments"
)

// ruleOverlong fails a function body whose comments outweigh its code.
const ruleOverlong = "overlong"

// maxWordsPerToken is the most comment words a function body may hold for each code token.
const maxWordsPerToken = 3

// minBodyBlocks is the fewest comment blocks a body needs before `overlong` judges it, since one comment may explain a no-op.
const minBodyBlocks = 2

// overlongErrors reports the body a block leads when its comments hold more than three words for each code token.
func overlongErrors(b comments.Block) []string {
	body := b.Body
	if !body.Lead || body.Blocks < minBodyBlocks || body.Words <= maxWordsPerToken*body.Tokens {
		return nil
	}
	return []string{fmt.Sprintf("%s:%d: %s: a function body holds %d comment words in %d comments for %d code tokens (max %d words per token); keep the comments that say why and cut the rest",
		b.Path, body.From, ruleOverlong, body.Words, body.Blocks, body.Tokens, maxWordsPerToken)}
}
