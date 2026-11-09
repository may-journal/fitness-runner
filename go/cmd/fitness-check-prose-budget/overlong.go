package check

import (
	"fmt"

	"github.com/may-journal/fitness-runner/go/internal/comments"
)

// ruleOverlong fails a function body whose comments outweigh its code.
const ruleOverlong = "overlong"

// overlongErrors reports the body a block leads when its comments hold more than lim.commentWordsPerTok words for each code token.
func overlongErrors(b comments.Block, lim limits) []string {
	body := b.Body
	if !body.Lead || body.Blocks < lim.bodyComments || body.Words <= lim.commentWordsPerTok*body.Tokens {
		return nil
	}
	return []string{fmt.Sprintf("%s:%d: %s: a function body holds %d comment words in %d comments for %d code tokens (max %d words per token); keep the comments that say why and cut the rest",
		b.Path, body.From, ruleOverlong, body.Words, body.Blocks, body.Tokens, lim.commentWordsPerTok)}
}
