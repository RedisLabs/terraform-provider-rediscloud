package pro

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUnitShouldWarnRedis8Modules_Redis8WithModules tests that warning is triggered for Redis 8.0 with modules
func TestUnitShouldWarnRedis8Modules_Redis8WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("8.0", true)
	assert.True(t, result, "should warn for Redis 8.0 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis80WithModules tests that warning is triggered for Redis 8.0.0 with modules
func TestUnitShouldWarnRedis8Modules_Redis80WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("8.0.0", true)
	assert.True(t, result, "should warn for Redis 8.0.0 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis81WithModules tests that warning is triggered for Redis 8.1+ with modules
func TestUnitShouldWarnRedis8Modules_Redis81WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("8.1.0", true)
	assert.True(t, result, "should warn for Redis 8.1.0 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis89WithModules tests that warning is triggered for Redis 8.9+ with modules
func TestUnitShouldWarnRedis8Modules_Redis89WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("8.9.9", true)
	assert.True(t, result, "should warn for Redis 8.9.9 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis7WithModules tests that no warning for Redis 7.x with modules
func TestUnitShouldWarnRedis8Modules_Redis7WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("7.4", true)
	assert.False(t, result, "should not warn for Redis 7.4 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis6WithModules tests that no warning for Redis 6.x with modules
func TestUnitShouldWarnRedis8Modules_Redis6WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("6.2", true)
	assert.False(t, result, "should not warn for Redis 6.2 with modules")
}

// TestUnitShouldWarnRedis8Modules_Redis8NoModules tests that no warning for Redis 8.0 without modules
func TestUnitShouldWarnRedis8Modules_Redis8NoModules(t *testing.T) {
	result := shouldWarnRedis8Modules("8.0", false)
	assert.False(t, result, "should not warn for Redis 8.0 without modules")
}

// TestUnitShouldWarnRedis8Modules_Redis7NoModules tests that no warning for Redis 7.x without modules
func TestUnitShouldWarnRedis8Modules_Redis7NoModules(t *testing.T) {
	result := shouldWarnRedis8Modules("7.4", false)
	assert.False(t, result, "should not warn for Redis 7.4 without modules")
}

// TestUnitShouldWarnRedis8Modules_Redis9WithModules tests that warning is triggered for Redis 9.x with modules (future-proofing)
func TestUnitShouldWarnRedis8Modules_Redis9WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("9.0", true)
	assert.True(t, result, "should warn for Redis 9.0 with modules (modules bundled in 8.0+)")
}

// TestUnitShouldWarnRedis8Modules_Redis10WithModules tests that warning is triggered for Redis 10.x with modules
func TestUnitShouldWarnRedis8Modules_Redis10WithModules(t *testing.T) {
	result := shouldWarnRedis8Modules("10.0.0", true)
	assert.True(t, result, "should warn for Redis 10.0.0 with modules (modules bundled in 8.0+)")
}

// TestUnitShouldSuppressModuleDiffsForRedis8_Redis8 tests that module diffs are suppressed for Redis 8.0
func TestUnitShouldSuppressModuleDiffsForRedis8_Redis8(t *testing.T) {
	result := shouldSuppressModuleDiffsForRedis8("8.0")
	assert.True(t, result, "should suppress module diffs for Redis 8.0")
}

// TestUnitShouldSuppressModuleDiffsForRedis8_Redis82 tests that module diffs are suppressed for Redis 8.2
func TestUnitShouldSuppressModuleDiffsForRedis8_Redis82(t *testing.T) {
	result := shouldSuppressModuleDiffsForRedis8("8.2")
	assert.True(t, result, "should suppress module diffs for Redis 8.2")
}

// TestUnitShouldSuppressModuleDiffsForRedis8_Redis9 tests that module diffs are suppressed for Redis 9.0
func TestUnitShouldSuppressModuleDiffsForRedis8_Redis9(t *testing.T) {
	result := shouldSuppressModuleDiffsForRedis8("9.0")
	assert.True(t, result, "should suppress module diffs for Redis 9.0")
}

// TestUnitShouldSuppressModuleDiffsForRedis8_Redis7 tests that module diffs are NOT suppressed for Redis 7.x
func TestUnitShouldSuppressModuleDiffsForRedis8_Redis7(t *testing.T) {
	result := shouldSuppressModuleDiffsForRedis8("7.4")
	assert.False(t, result, "should not suppress module diffs for Redis 7.4")
}

// TestUnitShouldSuppressModuleDiffsForRedis8_Redis6 tests that module diffs are NOT suppressed for Redis 6.x
func TestUnitShouldSuppressModuleDiffsForRedis8_Redis6(t *testing.T) {
	result := shouldSuppressModuleDiffsForRedis8("6.2")
	assert.False(t, result, "should not suppress module diffs for Redis 6.2")
}
