// Package docs provides OpenAPI/Swagger specifications for Delcom Blog REST API.
package docs

import (
	_ "embed"
	"encoding/json"

	"github.com/swaggo/swag"
)

//go:embed api.json
var ApiJSON []byte

// SwaggerInfo holds exported Swagger Info so clients can modify it.
var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Delcom Blog REST API",
	Description:      "Dokumentasi RESTful API Backend Delcom Blog (Multi-Tier & AI Generator dengan Search Engine) berbasis Golang Gin, Feature-First & TDD.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  string(ApiJSON),
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

// GetSwaggerJSON returns the embedded api.json with host and scheme dynamically set to match the current request URL.
func GetSwaggerJSON(host, scheme string) []byte {
	var doc map[string]interface{}
	if err := json.Unmarshal(ApiJSON, &doc); err != nil {
		return ApiJSON
	}
	if host != "" {
		doc["host"] = host
	}
	if scheme != "" {
		doc["schemes"] = []string{scheme}
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return ApiJSON
	}
	return updated
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
