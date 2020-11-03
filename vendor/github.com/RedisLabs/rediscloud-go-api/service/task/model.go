package task

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type Task struct {
	CommandType string    `json:"commandType"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Id          string    `json:"taskId"`
	Response    *Response `json:"response"`
}

type Response struct {
	Id       *int             `json:"resourceId"`
	Resource *json.RawMessage `json:"resource"`
	Error    *Error           `json:"error"`
}

type Error struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (e *Error) StatusCode() string {
	matches := errorStatusCode.FindStringSubmatch(e.Status)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s: %s", e.Status, e.Type, e.Description)
}

var errorStatusCode = regexp.MustCompile("^(\\d*).*$")
var _ error = &Error{}
