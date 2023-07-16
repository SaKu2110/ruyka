package rtc

import (
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

const (
	DISPATCH_KEYFRAME_INTERVAL           = 3 * time.Second
	SYNC_PEER_CONNECTIONS_RETRY_INTERVAL = 3 * time.Second
	SYNC_PEER_CONNECTIONS_ATTEMPT_LIMIT  = 25
)

type RTCEventType int

const (
	RTCEventTypeSyncSDP RTCEventType = iota
	RTCEventTypeAddTrack
	RTCEventTypeRemoveTrack
)

type RTCEventMessage struct {
	Event      RTCEventType
	LocalTrack *webrtc.TrackLocalStaticRTP
}

type TrackLocals map[string]*webrtc.TrackLocalStaticRTP

type TrackManager interface {
	Join(p PeerConnection) chan<- RTCEventMessage
}

type manager struct {
	mux         sync.RWMutex
	connections map[PeerConnectionID]PeerConnection
	trackLocals TrackLocals
}

func newTrackManager() TrackManager {
	m := &manager{
		mux:         sync.RWMutex{},
		connections: make(map[PeerConnectionID]PeerConnection),
		trackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
	}

	go m.dispatchKeyframeWorker()
	return m
}

func (m *manager) Join(p PeerConnection) chan<- RTCEventMessage {
	join := func() {
		m.mux.Lock()
		defer m.mux.Unlock()
		m.connections[p.ID()] = p
	}
	join()

	ch := make(chan RTCEventMessage)
	go m.rtcEventWorker(ch)
	return ch
}

func (m *manager) rtcEventWorker(ch chan RTCEventMessage) {
	defer close(ch)
	for {
		msg := <-ch
		switch msg.Event {
		case RTCEventTypeSyncSDP:
			zap.L().Info("rtc event worker: received sync sdp event")
			m.syncSessionDescriptionBetweenPeers()
		case RTCEventTypeAddTrack:
			m.addTrackLocal(msg.LocalTrack)
		case RTCEventTypeRemoveTrack:
			m.removeTrackLocal(msg.LocalTrack)
		default:
			zap.L().Warn("rtc event worker: received invalid message")
		}
	}
}

func (m *manager) dispatchKeyframe() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for id := range m.connections {
		m.connections[id].DispatchKeyframeRequest()
	}
}

func (m *manager) dispatchKeyframeWorker() {
	ticker := time.NewTicker(DISPATCH_KEYFRAME_INTERVAL)
	for range ticker.C {
		m.dispatchKeyframe()
	}
}

func (m *manager) syncSessionDescriptionBetweenPeers() {
	m.mux.Lock()
	defer func() {
		m.mux.Unlock()
		m.dispatchKeyframe()
	}()

	synk := func() bool {
		for id := range m.connections {
			connection := m.connections[id]
			if err := connection.UpdateTrack(m.trackLocals); err != nil {
				if err == ErrPeerConnClosed {
					delete(m.connections, id)
				}
				return false
			}
		}

		return true
	}
	retry := func() {
		time.Sleep(SYNC_PEER_CONNECTIONS_RETRY_INTERVAL)
		m.syncSessionDescriptionBetweenPeers()
	}

	for attempt := 0; ; attempt++ {
		if attempt == SYNC_PEER_CONNECTIONS_ATTEMPT_LIMIT {
			go retry()
			return
		}

		if success := synk(); success {
			break
		}
	}
}

func (m *manager) addTrackLocal(tr *webrtc.TrackLocalStaticRTP) {
	m.mux.Lock()
	defer func() {
		m.mux.Unlock()
		m.syncSessionDescriptionBetweenPeers()
	}()

	m.trackLocals[tr.ID()] = tr
}

func (m *manager) removeTrackLocal(tr *webrtc.TrackLocalStaticRTP) {
	m.mux.Lock()
	defer func() {
		m.mux.Unlock()
		m.syncSessionDescriptionBetweenPeers()
	}()

	delete(m.trackLocals, tr.ID())
}
