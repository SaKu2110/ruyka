package config

import (
	"net"
	"net/http"
	"ruyka/pkg/rtc"
	"ruyka/pkg/server"
	"ruyka/pkg/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Port        int           `yaml:"port,omitempty"`
	CORS        CORSConfig    `yaml:"cors,omitempty"`
	RTC         RTCConfig     `yaml:"rtc,omitempty"`
	Logging     LoggingConfig `yaml:"logging,omitempty"`
	Development bool          `yaml:"development,omitempty"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins,omitempty"`
	AllowHeaders []string `yaml:"allow_headers,omitempty"`
	AllowMethods []string `yaml:"allow_methods,omitempty"`
}

type RTCConfig struct {
	ICEServers []string     `yaml:"ice_servers,omitempty"`
	ICETCP     ICETCPConfig `yaml:"ice_tcp,omitempty"`
}

type ICETCPConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
	Port    int  `yaml:"port,omitempty"`
}

type LoggingConfig struct {
	zap.Config `yaml:",inline"`
}

var defaultConfig = Config{
	Port: 19000,
	CORS: CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	},
	RTC: RTCConfig{
		ICEServers: []string{"stun:stun.l.google.com:19302"},
		ICETCP: ICETCPConfig{
			Enabled: true,
			Port:    19443,
		},
	},
	Logging: LoggingConfig{
		zap.Config{
			Level: zap.NewAtomicLevelAt(zapcore.DebugLevel),
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Development: false,
			Encoding:    "json",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "msg",
				StacktraceKey:  "trace",
				FunctionKey:    "",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalLevelEncoder,
				EncodeTime:     zapcore.EpochTimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		},
	},
	Development: false,
}

func New() *Config {
	return &defaultConfig
}

func (c *Config) DevMode() {
	c.Logging.Config.Development = true
	c.Logging.Config.Encoding = "console"
	c.Logging.Config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	c.Development = true
}

func (c *Config) Build() (server.Server, error) {
	engine, err := c.buildEngine()
	if err != nil {
		return nil, err
	}
	logger, err := c.Logging.Build()
	if err != nil {
		return nil, err
	}
	rtcService, err := c.buildRTCService()
	if err != nil {
		return nil, err
	}

	return server.New(
		engine,
		logger,
		rtcService,
		c.Development,
	)
}

func (c *Config) buildEngine() (*echo.Echo, error) {
	addr := net.TCPAddr{IP: net.IP{0, 0, 0, 0}, Port: c.Port}
	if c.Development {
		addr.IP = net.IP{127, 0, 0, 1}
	}
	listener, err := net.Listen("tcp", addr.String())
	if err != nil {
		return nil, err
	}
	cors := middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: c.CORS.AllowOrigins,
		AllowHeaders: c.CORS.AllowHeaders,
		AllowMethods: c.CORS.AllowMethods,
	}

	e := echo.New()
	e.Listener = listener
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(cors))
	return e, nil
}

func (c *Config) buildRTCService() (service.Service, error) {
	i := &interceptor.Registry{}
	m, err := rtc.NewMediaEngine()
	if err != nil {
		return nil, err
	}
	s, err := c.buildSettingEngine()
	if err != nil {
		return nil, err
	}

	r, err := rtc.NewAPI(s, m, i, &webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: c.RTC.ICEServers},
		},
	})
	if err != nil {
		return nil, err
	}
	svc := service.NewRTCService(r)
	return svc, nil
}

func (c *Config) buildSettingEngine() (*webrtc.SettingEngine, error) {
	s := &webrtc.SettingEngine{}
	if !c.RTC.ICETCP.Enabled {
		return s, nil
	}

	addr := &net.TCPAddr{IP: net.IP{0, 0, 0, 0}, Port: c.RTC.ICETCP.Port}
	lis, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return s, err
	}

	// setup SettingEngine
	s.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeTCP6,
	})
	s.SetICETCPMux(
		webrtc.NewICETCPMux(nil, lis, 8),
	)
	return s, nil
}
