package rtc

import (
	"github.com/pion/webrtc/v3"
)

func NewMediaEngine() (m *webrtc.MediaEngine, err error) {
	var extensions = []string{
		"urn:ietf:params:rtp-hdrext:sdes:mid",
		"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
		"urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
	}

	m = &webrtc.MediaEngine{}
	if err = m.RegisterDefaultCodecs(); err != nil {
		return
	}
	for _, uri := range extensions {
		if err = m.RegisterHeaderExtension(
			webrtc.RTPHeaderExtensionCapability{URI: uri},
			webrtc.RTPCodecTypeVideo,
		); err != nil {
			return
		}
	}
	return
}
