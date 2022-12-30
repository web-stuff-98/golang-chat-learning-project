package mylimiter

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type IpInfo struct {
	lastRequest      time.Time
	requestsInWindow uint8
}

type SimpleLimiterOpts struct {
	Window        time.Duration
	MaxReqs       uint8
	BlockDuration time.Duration
	Message       string
}

func SimpleLimiterMiddleware(ipInfoMap map[string]IpInfo, opts SimpleLimiterOpts) fiber.Handler {
	return func(c *fiber.Ctx) error {
		info, ok := ipInfoMap[c.IP()]
		if ok {
			// First check if blocked
			if info.requestsInWindow >= opts.MaxReqs {
				if time.Now().After(info.lastRequest.Add(opts.BlockDuration)) {
					// The IP was blocked, but is now no longer blocked
					delete(ipInfoMap, c.IP())
					return c.Next()
				} else {
					// The IP is still waiting for the end of the block duration
					var msg string
					if opts.Message != "" {
						msg = opts.Message
					} else {
						msg = "Too many requests"
					}
					c.Status(fiber.StatusTooManyRequests)
					return c.JSON(fiber.Map{
						"message": msg,
					})
				}
			}
			// If not blocked add to the number of requests
			if info.lastRequest.Before(time.Now().Add(-opts.Window)) {
				info.requestsInWindow = 1
			} else {
				info.requestsInWindow += 1
			}
			info.lastRequest = time.Now()
		} else {
			ipInfoMap[c.IP()] = IpInfo{
				lastRequest: time.Now(),
			}
		}
		return c.Next()
	}
}
