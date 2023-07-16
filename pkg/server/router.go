package server

import (
	"ruyka/pkg/service"

	"github.com/labstack/echo/v4"
)

func route(
	e *echo.Echo,
	rtcService service.Service,
) error {
	apiv1 := e.Group("api/v1")
	apiv1.GET("/signaling", rtcService.Serve())

	return nil
}
