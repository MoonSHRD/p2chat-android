package match

import "github.com/libp2p/go-libp2p-core/peer"

// Response alias is used for json marshaling
// the match ([topic]=>[mxIDs]) response
type Response map[string][]peer.ID
