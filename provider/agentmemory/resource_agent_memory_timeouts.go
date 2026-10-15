package agentmemory

import "time"

// Bound SDK polling and HTTP work even when Terraform supplies no deadline.
// A shorter caller deadline or cancellation always takes precedence. These
// internal limits do not add schema fields or change existing state formats.
const (
	agentMemoryMutationTimeout = 30 * time.Minute
	agentMemoryReadTimeout     = 5 * time.Minute
	agentMemoryAPIKeyTimeout   = 5 * time.Minute
)
