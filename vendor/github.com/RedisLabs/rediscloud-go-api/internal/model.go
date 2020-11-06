package internal

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/RedisLabs/rediscloud-go-api/redis"
)

type task struct {
	CommandType *string   `json:"commandType,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	ID          *string   `json:"taskId,omitempty"`
	Response    *response `json:"response,omitempty"`
}

func (o task) String() string {
	return ToString(o)
}

type response struct {
	ID       *int             `json:"resourceId,omitempty"`
	Resource *json.RawMessage `json:"resource,omitempty"`
	Error    *Error           `json:"error,omitempty"`
}

func (o response) String() string {
	return ToString(o)
}

type Error struct {
	Type        *string `json:"type,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (o Error) String() string {
	return ToString(o)
}

func (e *Error) StatusCode() string {
	matches := errorStatusCode.FindStringSubmatch(redis.StringValue(e.Status))
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s: %s", redis.StringValue(e.Status), redis.StringValue(e.Type), redis.StringValue(e.Description))
}

var errorStatusCode = regexp.MustCompile("^(\\d*).*$")
var _ error = &Error{}
