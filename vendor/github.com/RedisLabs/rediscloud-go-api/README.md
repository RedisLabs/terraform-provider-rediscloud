# rediscloud-go-api

This repository is a Go SDK for the [Redis Cloud REST API](https://docs.redislabs.com/latest/rc/api/).

## Getting Started

### Installing
You can use this module by using `go get` to add it to either your `GOPATH` workspace or
the project's dependencies.
```shell script
go get github.com/RedisLabs/rediscloud-go-api
```

### Example
This is an example of using the SDK
```go
package main

import (
	"context"
	"fmt"

	rediscloud_api "github.com/RedisLabs/rediscloud-go-api"
)

func main() {
	// The client will use the credentials from `REDISCLOUD_API_KEY` and `REDISCLOUD_SECRET_KEY` by default
	client, err := rediscloud_api.NewClient()
	if err != nil {
		panic(err)
	}

	task, err := client.Task.Get(context.TODO(), "task-uuid")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found task: %#v", task)
}
```
