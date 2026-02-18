package requestctx

import "github.com/pocketbase/pocketbase/core"

func String(c *core.RequestEvent, key string) string {
	if c == nil {
		return ""
	}
	value, _ := c.Get(key).(string)
	return value
}
