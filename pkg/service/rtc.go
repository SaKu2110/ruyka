package service

import (
	"ruyka/pkg/rtc"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

type rtcService struct {
	rtc      rtc.RTC
	upgrader websocket.Upgrader
}

func NewRTCService(
	r rtc.RTC,
) Service {
	const (
		WebSocketHandshakeTimeout = 30 * time.Second
		WebSocketReadBufferSize   = 1024
		WebSocketWriteBufferSize  = 1024
	)
	return &rtcService{
		rtc: r,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: WebSocketHandshakeTimeout,
			ReadBufferSize:   WebSocketReadBufferSize,
			WriteBufferSize:  WebSocketWriteBufferSize,
		},
	}
}

func (s *rtcService) Serve() echo.HandlerFunc {
	type message struct {
		Event              rtc.EventType                    `json:"event"`
		SessionDescription rtc.SessionDescriptionSerializer `json:"sdp,omitempty"`
		ICECandidate       rtc.ICECandidateSerializer       `json:"ice,omitempty"`
	}
	return func(cxt echo.Context) error {
		if !websocket.IsWebSocketUpgrade(cxt.Request()) {
			return nil
		}
		c, err := s.upgrader.Upgrade(cxt.Response(), cxt.Request(), nil)
		if err != nil {
			return err
		}
		defer c.Close()

		sc := rtc.NewSignalConnection(c)
		peer, err := s.rtc.NewPeerConnection(sc)
		if err != nil {
			return err
		}
		defer peer.Close()

		zap.L().Info("new peer connection joined")
		for {
			msg := message{}
			if err := sc.ReadMessage(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure,
				) {
					zap.L().Warn(err.Error())
				} else {
					zap.L().Info("websocket connection closed")
				}
				return nil
			}

			switch msg.Event {
			case rtc.EventTypeAnswer:
				if err := peer.UpdateRemoteDescription(msg.SessionDescription); err != nil {
					zap.L().Warn(err.Error())
					return err
				}
			case rtc.EventTypeCandidate:
				if err := peer.UpdateICECandidate(msg.ICECandidate); err != nil {
					zap.L().Warn(err.Error())
					return err
				}
			default:
				return nil
			}
		}
	}
}
