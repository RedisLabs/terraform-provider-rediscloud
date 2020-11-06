package internal

import (
	"encoding/json"
	"fmt"
)

func ToString(o interface{}) string {
	output, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprintf("%#v", o)
	}
	return string(output)
}
