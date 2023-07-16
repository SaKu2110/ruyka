package rtc

import (
	"errors"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

var (
	ErrPeerConnClosed = errors.New("peer connection is already closed")
)

type PeerConnectionID xid.ID

type PeerConnection interface {
	Close() error
	DispatchKeyframeRequest()
	ID() PeerConnectionID
	UpdateLocalDescription() (SessionDescriptionSerializer, error)
	UpdateRemoteDescription(SessionDescriptionSerializer) error
	UpdateICECandidate(ICECandidateSerializer) error
	UpdateTrack(TrackLocals) error
}

type connection struct {
	id   PeerConnectionID
	conn SignalConnection
	peer *webrtc.PeerConnection
}

func newPeerConnection(
	c SignalConnection,
	p *webrtc.PeerConnection,
	m TrackManager,
) (PeerConnection, error) {
	conn := &connection{
		id:   PeerConnectionID(xid.New()),
		conn: c,
		peer: p,
	}
	return conn, nil
}

func (c *connection) Close() error {
	return c.peer.Close()
}

func (c *connection) DispatchKeyframeRequest() {
	// FIXME: replace to dispatch FIR
	c.dispatchPLIToReceivers()
}

func (c *connection) dispatchOffer() error {
	type message struct {
		Event              EventType                    `json:"event"`
		SessionDescription SessionDescriptionSerializer `json:"sdp,omitempty"`
	}

	sdp, err := c.UpdateLocalDescription()
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(message{
		Event:              EventTypeOffer,
		SessionDescription: sdp,
	})
}

func (c *connection) ID() PeerConnectionID {
	return c.id
}

func (c *connection) UpdateLocalDescription() (SessionDescriptionSerializer, error) {
	s := SessionDescriptionSerializer{}
	offer, err := c.peer.CreateOffer(&webrtc.OfferOptions{})
	if err != nil {
		return s, err
	}

	if err := c.peer.SetLocalDescription(offer); err != nil {
		return s, err
	}
	s.SessionDescription = offer
	return s, err
}

func (c *connection) UpdateRemoteDescription(
	desc SessionDescriptionSerializer,
) error {
	zap.L().Info("set remote description")
	return c.peer.SetRemoteDescription(desc.SessionDescription)
}

func (c *connection) UpdateICECandidate(
	serializer ICECandidateSerializer,
) error {
	zap.L().Info("add ice candidate")
	return c.peer.AddICECandidate(serializer.ICECandidateInit)
}

func (c *connection) UpdateTrack(tracks TrackLocals) error {
	state := c.peer.ConnectionState()
	if state == webrtc.PeerConnectionStateClosed {
		return ErrPeerConnClosed
	}

	m := map[string]bool{}
	for _, sender := range c.peer.GetSenders() {
		if sender.Track() == nil {
			continue
		}

		m[sender.Track().ID()] = true
		if _, ok := tracks[sender.Track().ID()]; !ok {
			if err := c.peer.RemoveTrack(sender); err != nil {
				return err
			}
		}
	}

	for _, receiver := range c.peer.GetReceivers() {
		if receiver.Track() == nil {
			continue
		}

		m[receiver.Track().ID()] = true
	}

	for id, track := range tracks {
		if _, ok := m[id]; !ok {
			if _, err := c.peer.AddTrack(track); err != nil {
				return err
			}
		}
	}
	return c.dispatchOffer()
}

//lint:ignore U1000 PLIとFIRを使い分けるようにするまで一旦、定義したままにする
func (c *connection) dispatchFIRToReceivers() {
	for _, receiver := range c.peer.GetReceivers() {
		track := receiver.Track()
		if track == nil {
			continue
		}

		pkts := []rtcp.Packet{
			&rtcp.FullIntraRequest{
				MediaSSRC: uint32(track.SSRC()),
			},
		}

		if err := c.peer.WriteRTCP(pkts); err != nil {
			continue
		}
	}
}

func (c *connection) dispatchPLIToReceivers() {
	for _, receiver := range c.peer.GetReceivers() {
		track := receiver.Track()
		if track == nil {
			continue
		}

		pkts := []rtcp.Packet{
			&rtcp.PictureLossIndication{
				MediaSSRC: uint32(track.SSRC()),
			},
		}

		if err := c.peer.WriteRTCP(pkts); err != nil {
			continue
		}
	}
}
