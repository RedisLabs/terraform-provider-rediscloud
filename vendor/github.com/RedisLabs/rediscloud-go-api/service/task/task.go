// Package task allows the interaction task resource. Tasks are created by other resources to execute various actions
// that can take significant about of time. All APIs within this package will eventually be migrated to `internal/`
// once the other resources have been implemented in the SDK, as these other resources will provide an interface
// which wraps this package.
package task
