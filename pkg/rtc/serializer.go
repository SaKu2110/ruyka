package rtc

import (
	"github.com/pion/webrtc/v3"
)

// ref: https://pkg.go.dev/github.com/pion/webrtc/v3@v3.2.12#ICECandidateInit
type ICECandidateSerializer struct {
	webrtc.ICECandidateInit `json:",inline"`
}

type SessionDescriptionSerializer struct {
	webrtc.SessionDescription `json:",inline"`
}
