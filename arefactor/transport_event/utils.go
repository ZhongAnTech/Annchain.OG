package transport_event

import (
	"github.com/annchain/OG/arefactor/common/files"
	"strings"
)

func PrettyId(peerId string) string {
	l := len(peerId)
	if l > 10 {
		l = 10
	}
	return peerId[0:l]
}

func PrettyIds(peerId []string) string {
	s := make([]string, len(peerId))
	for i, v := range peerId {
		l := len(v)
		if l > 10 {
			l = 10
		}
		s[i] = v[0:l]
	}
	return strings.Join(s, ",")
}

func LoadKnownPeers(path string) (peers []string, err error) {
	return files.ReadLines(path)
}
