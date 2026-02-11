package health

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func Health(c *core.RequestEvent) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
