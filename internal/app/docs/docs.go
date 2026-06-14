package docs

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func RegisterRoutes(app *fiber.App) {
	app.Get("/docs/*", static.New("./docs"))

	app.Get("/docs", func(c fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(`<!DOCTYPE html>
<html>
<head>
  <title>CentraChannel API - Redoc</title>
  <meta charset="utf-8"/>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</head>
<body>
  <div id="redoc-container"></div>
  <script>
    Redoc.init('/docs/api/openapi.json', {}, document.getElementById('redoc-container'))
  </script>
</body>
</html>`)
	})

	app.Get("/swagger", func(c fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(`<!DOCTYPE html>
<html>
<head>
  <title>CentraChannel API - Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/docs/api/openapi.json',
      dom_id: '#swagger-ui'
    })
  </script>
</body>
</html>`)
	})
}
