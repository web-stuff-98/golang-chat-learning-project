package mylimiter

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type BlockInfo struct {
	LastRequest      time.Time
	RequestsInWindow uint8
	OptsUsed         SimpleLimiterOpts
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
				if routeInfo.RequestsInWindow >= opts.MaxReqs {
					if time.Now().After(routeInfo.LastRequest.Add(opts.BlockDuration)) {
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
				// If not blocked add to the number of requests
				if routeInfo.LastRequest.Before(time.Now().Add(-opts.Window)) {
					routeInfo.RequestsInWindow = 1
				} else {
					routeInfo.RequestsInWindow += 1
				}
				routeInfo.LastRequest = time.Now()
				ipInfoMap[c.IP()][opts.RouteName] = routeInfo
			} else {
				innerMap := make(map[string]BlockInfo)
				innerMap[opts.RouteName] = BlockInfo{
					LastRequest:      time.Now(),
					RequestsInWindow: 1,
					OptsUsed:         opts,
				}
				ipInfoMap[c.IP()] = innerMap
			}
		} else {
			innerMap := make(map[string]BlockInfo)
			innerMap[opts.RouteName] = BlockInfo{
				LastRequest:      time.Now(),
				RequestsInWindow: 1,
				OptsUsed:         opts,
			}
			ipInfoMap[c.IP()] = innerMap
		}
		return c.Next()
	}
}
