package redis

import "time"

func Int(i int) *int {
	return &i
}

func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func String(s string) *string {
	return &s
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func Float64(f float64) *float64 {
	return &f
}

func Float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func Bool(b bool) *bool {
	return &b
}

func BoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func Time(d time.Time) *time.Time {
	return &d
}

func TimeValue(d *time.Time) time.Time {
	if d == nil {
		return time.Time{}
	}
	return *d
}

func StringSlice(ss ...string) []*string {
	var ret []*string
	for _, s := range ss {
		ret = append(ret, String(s))
	}
	return ret
}

func StringSliceValue(ss ...*string) []string {
	var ret []string
	for _, s := range ss {
		if s != nil {
			ret = append(ret, StringValue(s))
		}
	}
	return ret
}
