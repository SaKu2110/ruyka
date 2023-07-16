package rtc

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type RTC interface {
	NewPeerConnection(SignalConnection) (PeerConnection, error)
}

type rtc struct {
	api   *webrtc.API
	conf  *webrtc.Configuration
	track TrackManager
}

func NewAPI(
	s *webrtc.SettingEngine,
	m *webrtc.MediaEngine,
	i *interceptor.Registry,
	c *webrtc.Configuration,
) (RTC, error) {
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	return &rtc{
		api: webrtc.NewAPI(
			webrtc.WithSettingEngine(*s),
			webrtc.WithMediaEngine(m),
			webrtc.WithInterceptorRegistry(i),
		),
		conf:  c,
		track: newTrackManager(),
	}, nil
}

func (r *rtc) NewPeerConnection(
	sc SignalConnection,
) (PeerConnection, error) {
	p, err := r.api.NewPeerConnection(*r.conf)
	if err != nil {
		return nil, err
	}
	peer, err := newPeerConnection(sc, p, r.track)
	if err != nil {
		return nil, err
	}
	ch := r.track.Join(peer)

	setup := func(p *webrtc.PeerConnection) error {
		type message struct {
			Event        EventType              `json:"event"`
			ICECandidate ICECandidateSerializer `json:"ice,omitempty"`
		}

		if _, err := p.AddTransceiverFromKind(
			webrtc.RTPCodecTypeAudio,
			webrtc.RTPTransceiverInit{
				Direction:     webrtc.RTPTransceiverDirectionRecvonly,
				SendEncodings: []webrtc.RTPEncodingParameters{},
			},
		); err != nil {
			return err
		}
		if _, err := p.AddTransceiverFromKind(
			webrtc.RTPCodecTypeVideo,
			webrtc.RTPTransceiverInit{
				Direction:     webrtc.RTPTransceiverDirectionRecvonly,
				SendEncodings: []webrtc.RTPEncodingParameters{},
			},
		); err != nil {
			return err
		}

		p.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
			tl, err := webrtc.NewTrackLocalStaticRTP(
				tr.Codec().RTPCodecCapability,
				tr.ID(),
				tr.StreamID(),
			)
			if err != nil {
				zap.L().Warn("on track: failed new track local static rtp")
				return
			}
			ch <- RTCEventMessage{Event: RTCEventTypeAddTrack, LocalTrack: tl}
			defer func() {
				ch <- RTCEventMessage{Event: RTCEventTypeRemoveTrack, LocalTrack: tl}
			}()

			for {
				pkt, _, err := tr.ReadRTP()
				if err != nil {
					return
				}
				if err := tl.WriteRTP(pkt); err != nil {
					return
				}
			}
		})
		p.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
			switch pcs {
			case webrtc.PeerConnectionStateConnected:
				ch <- RTCEventMessage{Event: RTCEventTypeSyncSDP}
			case webrtc.PeerConnectionStateClosed:
				ch <- RTCEventMessage{Event: RTCEventTypeSyncSDP}
			case webrtc.PeerConnectionStateFailed:
				p.Close()
			}
		})
		p.OnICECandidate(func(i *webrtc.ICECandidate) {
			if i == nil {
				return
			}

			zap.L().Info("on ice candidate: send message to client")
			err := sc.WriteMessage(message{
				Event: EventTypeCandidate,
				ICECandidate: ICECandidateSerializer{
					ICECandidateInit: i.ToJSON(),
				},
			})
			if err != nil {
				zap.L().Warn(err.Error())
			}
		})
		return nil
	}

	if err := setup(p); err != nil {
		p.Close()
		return nil, err
	}

	ch <- RTCEventMessage{Event: RTCEventTypeSyncSDP}
	return peer, nil
}
