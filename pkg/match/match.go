package match

import (
	"github.com/MoonSHRD/p2chat-android/pkg/utils"
)

// Response alias is used for json marshaling
// the whole match ([topic]=>[matrixIDs]) response
// or for the new matches ([matrixID]=>[topics])
type Response map[string][]string

// MatchProcessor is helper type for work
// with match operations processing
type MatchProcessor struct {
	mathes          Response
	newMatchesQueue utils.Queue
}

// GetAllMatches returns the whole matches map within json format
func (mc *MatchProcessor) GetAllMatches() string {
	return utils.ObjectToJSON(mc.mathes)
}

// GetNewMatch returns new match from the matches-queue
func (mc *MatchProcessor) GetNewMatch() string {
	return utils.ObjectToJSON(mc.newMatchesQueue.PopBack())
}

// AddNewMatch pushes new match [topic]=>[newMatrixID] to the matches-queue and map
func (mc *MatchProcessor) AddNewMatch(matrixID string, topics []string) {
	for _, topic := range topics {
		mc.mathes[topic] = append(mc.mathes[topic], matrixID)
	}
	mc.newMatchesQueue.PushBack(Response{matrixID: topics})
}
