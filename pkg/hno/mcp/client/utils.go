package client

import (
	"encoding/json"
)

// parseResult unmarshals JSON raw message into the target interface
// parseResult 将 JSON 原始消息解析为目标接口
func parseResult(data json.RawMessage, target interface{}) error {
	return json.Unmarshal(data, target)
}
