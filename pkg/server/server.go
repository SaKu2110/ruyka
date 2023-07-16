package server

import (
	"context"
	"os"
	"os/signal"
	"ruyka/app"
	"ruyka/pkg/service"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const gracefulShutdownTimeout = 10 * time.Second

type Server interface {
	Run()
	Shutdown() error
}

type server struct {
	exit   chan os.Signal
	engine *echo.Echo
	logger *zap.Logger
}

func New(
	engine *echo.Echo,
	logger *zap.Logger,
	rtcService service.Service,
	isDevelopment bool,
) (Server, error) {
	if err := route(
		engine,
		rtcService,
	); err != nil {
		return nil, err
	}
	if isDevelopment {
		err := app.Router(engine)
		if err != nil {
			return nil, err
		}
	}
	return &server{
		exit:   make(chan os.Signal, 1),
		engine: engine,
		logger: logger,
	}, nil
}

func (s *server) Run() {
	repair := zap.ReplaceGlobals(s.logger)
	defer func() {
		s.logger.Sync()
		repair()
	}()

	if err := s.engine.Start(""); err != nil {
		zap.L().Fatal(err.Error())
	}
}

func (s *server) Shutdown() error {
	signal.Notify(s.exit, os.Interrupt)

	<-s.exit
	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	return s.engine.Shutdown(ctx)
}
