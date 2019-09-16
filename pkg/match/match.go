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

// GetAllMatches returns the whole matches map within json format (or empty string, if we haven't any matches)
func (mp *MatchProcessor) GetAllMatches() string {
	if len(mp.mathes) != 0 {
		return utils.ObjectToJSON(mp.mathes)
	}
	return ""
}

// GetNewMatch returns new match from the matches-queue
func (mp *MatchProcessor) GetNewMatch() string {
	newMatch := mp.newMatchesQueue.PopBack()
	// if we have new match, then jsonify it
	if newMatch != nil {
		return utils.ObjectToJSON(newMatch)
	}
	// else just return empty string
	return ""
}

// AddNewMatch pushes new match [topic]=>[newMatrixID] to the matches-queue and map
func (mp *MatchProcessor) AddNewMatch(matrixID string, topics []string) {
	for _, topic := range topics {
		mp.mathes[topic] = append(mp.mathes[topic], matrixID)
	}
	mp.newMatchesQueue.PushBack(Response{matrixID: topics})
}
