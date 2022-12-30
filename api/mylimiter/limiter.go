package mylimiter

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type BlockInfo struct {
	lastRequest      time.Time
	requestsInWindow uint8
}

type SimpleLimiterOpts struct {
	Window        time.Duration
	MaxReqs       uint8
	BlockDuration time.Duration
	Message       string
	RouteName     string
}

func SimpleLimiterMiddleware(ipInfoMap map[string]map[string]BlockInfo, opts SimpleLimiterOpts) fiber.Handler {
	return func(c *fiber.Ctx) error {
		info, ipInfoOk := ipInfoMap[c.IP()]
		routeInfo, routeInfoOk := info[opts.RouteName]
		if ipInfoOk {
			if routeInfoOk {
				// First check if blocked
				if routeInfo.requestsInWindow >= opts.MaxReqs {
					if time.Now().After(routeInfo.lastRequest.Add(opts.BlockDuration)) {
						// The IP was blocked, but is now no longer blocked
						delete(ipInfoMap[c.IP()], opts.RouteName)
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
			}
			if routeInfoOk {
				// If not blocked add to the number of requests
				if routeInfo.lastRequest.Before(time.Now().Add(-opts.Window)) {
					routeInfo.requestsInWindow = 1
				} else {
					routeInfo.requestsInWindow += 1
				}
				routeInfo.lastRequest = time.Now()
				ipInfoMap[c.IP()][opts.RouteName] = routeInfo
			} else {
				ipInfoMap[c.IP()][opts.RouteName] = BlockInfo{
					lastRequest:      time.Now(),
					requestsInWindow: 1,
				}
			}
		} else {
			ipInfoMap[c.IP()][opts.RouteName] = BlockInfo{
				lastRequest: time.Now(),
			}
		}
		return c.Next()
	}
}
