package agentmemory

import "time"

const (
	agentMemoryMutationDefaultTimeout = 30 * time.Minute
	agentMemoryReadDefaultTimeout     = 5 * time.Minute
	agentMemoryAPIKeyDefaultTimeout   = 5 * time.Minute
)
