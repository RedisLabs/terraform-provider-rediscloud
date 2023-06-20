package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
)

type debugLogger struct{}

func (d *debugLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	message := fmt.Sprintf("[rediscloud-go-api] "+format, v...)
	tflog.Debug(ctx, message)
}

func (d *debugLogger) Println(ctx context.Context, v ...interface{}) {
	var items []string
	for _, i := range v {
		items = append(items, fmt.Sprintf("%s", i))
	}
	tflog.Debug(ctx, fmt.Sprintf("[rediscloud-go-api] %s", strings.Join(items, " ")))
}
