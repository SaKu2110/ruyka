package service

import "github.com/labstack/echo/v4"

type Service interface {
	Serve() echo.HandlerFunc
}
