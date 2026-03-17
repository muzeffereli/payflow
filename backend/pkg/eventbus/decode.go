package eventbus

import (
	"encoding/json"
	"fmt"
)

func DecodeData(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode event data: %w", err)
	}
	return nil
}
