package app

import (
	"fmt"
	"html/template"
	"os"

	"github.com/labstack/echo/v4"
)

const baseURL = "ruyka/meet"

var (
	SIGNALING_API_URL_FORMAT = "ws://%s/api/v1/signaling"
)

func Router(engine *echo.Echo) error {
	app := engine.Group(baseURL)
	{
		text, err := os.ReadFile("app/index.js")
		if err != nil {
			return err
		}

		tmpl := template.Must(template.New("").Parse(string(text)))
		app.GET("/index.js", func(ctx echo.Context) error {
			return tmpl.Execute(
				ctx.Response().Writer,
				fmt.Sprintf(SIGNALING_API_URL_FORMAT, ctx.Request().Host),
			)
		})
		app.File("", "app/index.html")
		app.Static("", "app")
	}
	return nil
}
