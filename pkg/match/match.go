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
func (mp *MatchProcessor) GetAllMatches() string {
	return utils.ObjectToJSON(mp.mathes)
}

// GetNewMatch returns new match from the matches-queue
func (mp *MatchProcessor) GetNewMatch() string {
	return utils.ObjectToJSON(mp.newMatchesQueue.PopBack())
}

// AddNewMatch pushes new match [topic]=>[newMatrixID] to the matches-queue and map
func (mp *MatchProcessor) AddNewMatch(matrixID string, topics []string) {
	for _, topic := range topics {
		mp.mathes[topic] = append(mp.mathes[topic], matrixID)
	}
	mp.newMatchesQueue.PushBack(Response{matrixID: topics})
}
