package match

import (
	"github.com/MoonSHRD/p2chat-android/pkg/utils"
)

type Match struct {
	PeerID   string `json:"peerID"`
	MatrixID string `json:"matrixID"`
	Topic    string `json:"topic"`
	IsValid  bool   `json:"isValid"`
}

// MatchProcessor is helper type for work
// with match operations processing
type MatchProcessor struct {
	mathes          map[string][]Match
	newMatchesQueue *utils.Queue
}

func NewMatchProcessor() *MatchProcessor {
	return &MatchProcessor{
		mathes:          make(map[string][]Match),
		newMatchesQueue: utils.New(),
	}
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
func (mp *MatchProcessor) AddNewMatch(topic string, peerID string, matrixID string) {
	match := Match{
		PeerID:   peerID,
		MatrixID: matrixID,
		Topic:    topic,
		IsValid:  true,
	}
	mp.mathes[topic] = append(mp.mathes[topic], match)
	mp.newMatchesQueue.PushBack(match)
}

func (mp *MatchProcessor) RemoveMatch(topic string, peerID string, matrixID string) {
	for i := 0; i < len(mp.mathes[topic]); i++ {
		if mp.mathes[topic][i].PeerID == peerID && mp.mathes[topic][i].MatrixID == matrixID {
			mp.removeMatch(topic, i)
			break
		}
	}

	invalidMatchEvent := Match{
		PeerID:   peerID,
		MatrixID: matrixID,
		Topic:    topic,
		IsValid:  false,
	}
	mp.newMatchesQueue.PushBack(invalidMatchEvent)
}

func (mp *MatchProcessor) removeMatch(topic string, index int) {
	mp.mathes[topic] = append(mp.mathes[topic][:index], mp.mathes[topic][index+1:]...)
}
