package docs

import _ "embed"

// OpenAPI is the backend contract exposed by /swagger/openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte

const SwaggerHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DAVEX DID Backend API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>SwaggerUIBundle({url: "/swagger/openapi.yaml", dom_id: "#swagger-ui", persistAuthorization: true});</script>
</body>
</html>`
