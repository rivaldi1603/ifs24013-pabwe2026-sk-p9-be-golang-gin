package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSwaggerJSON(t *testing.T) {
	data := GetSwaggerJSON("localhost:8081", "https")
	var doc map[string]interface{}
	err := json.Unmarshal(data, &doc)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8081", doc["host"])
	assert.Equal(t, []interface{}{"https"}, doc["schemes"])

	dataEmpty := GetSwaggerJSON("", "")
	var docEmpty map[string]interface{}
	errEmpty := json.Unmarshal(dataEmpty, &docEmpty)
	assert.NoError(t, errEmpty)
	assert.Equal(t, "", docEmpty["host"])

	origApiJSON := ApiJSON
	defer func() { ApiJSON = origApiJSON }()
	ApiJSON = []byte("invalid json")
	fallback := GetSwaggerJSON("api.test", "http")
	assert.Equal(t, []byte("invalid json"), fallback)
}
