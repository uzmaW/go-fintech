package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func PrometheusMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		m.IncInFlight()

		c.Next()

		m.DecInFlight()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		m.ObserveHTTPRequest(c.Request.Method, path, statusCode, duration)
	}
}

func DBQueryTracker(m *Metrics) func(operation string, fn func() error) error {
	return func(operation string, fn func() error) error {
		start := time.Now()
		err := fn()
		duration := time.Since(start)

		m.ObserveDBQuery(operation, duration)

		if err != nil {
			m.IncErrors("db_" + operation)
		}

		return err
	}
}

func StatusFromInt(code int) string {
	return strconv.Itoa(code)
}
