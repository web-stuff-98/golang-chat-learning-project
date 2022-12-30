package routes

import (
	"time"

	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/mylimiter"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, chatServer *controllers.ChatServer, closeWsChan chan string, protectedUids map[primitive.ObjectID]struct{}, protectedRids map[primitive.ObjectID]struct{}, ipBlockInfoMap map[string]mylimiter.IpInfo) {
	app.Post("/api/welcome", controllers.Welcome)
	app.Post("/api/user/login", controllers.HandleLogin)
	app.Post("/api/user/register", controllers.HandleRegister)
	app.Post("/api/user/updatepfp", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       10,
		BlockDuration: time.Second * 30,
	}), helpers.AuthMiddleware, controllers.HandleUpdatePfp(chatServer, protectedUids))
	app.Post("/api/user/deleteacc", helpers.AuthMiddleware, controllers.HandleDeleteUser(protectedUids))
	app.Post("/api/user/refresh", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 120,
		MaxReqs:       4,
		BlockDuration: time.Minute * 2,
	}), controllers.HandleRefresh(closeWsChan))
	app.Post("/api/user/logout", controllers.HandleLogout(closeWsChan))
	app.Get("/api/user/:id", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       30,
		BlockDuration: time.Second * 4,
	}), helpers.AuthMiddleware, controllers.HandleGetUser)

	app.Use("/ws", controllers.HandleWsUpgrade)
	app.Get("/ws/conn", controllers.HandleWsConn(chatServer, closeWsChan))

	app.Get("/api/room/:id", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       10,
		BlockDuration: time.Second * 30,
	}), helpers.AuthMiddleware, controllers.HandleGetRoom)
	app.Get("/api/rooms", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 3,
		MaxReqs:       5,
		BlockDuration: time.Second * 100,
	}), helpers.AuthMiddleware, controllers.HandleGetRooms)
	app.Patch("/api/room/:id", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       4,
		BlockDuration: time.Second * 30,
	}), helpers.AuthMiddleware, controllers.HandleUpdateRoom(protectedRids))
	app.Delete("/api/room/:id", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 3,
		MaxReqs:       4,
		BlockDuration: time.Second * 30,
	}), helpers.AuthMiddleware, controllers.HandleDeleteRoom(chatServer, protectedRids))
	app.Post("/api/room/:id/image", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       5,
		BlockDuration: time.Minute,
	}), helpers.AuthMiddleware, controllers.HandleUploadRoomImage(chatServer))
	app.Post("/api/room/:id/join", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       10,
		BlockDuration: time.Second * 10,
	}), helpers.AuthMiddleware, controllers.HandleJoinRoom(chatServer))
	app.Post("/api/room/:id/leave", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Second * 10,
		MaxReqs:       10,
		BlockDuration: time.Second * 10}), helpers.AuthMiddleware, controllers.HandleLeaveRoom(chatServer))
	app.Post("/api/room", mylimiter.SimpleLimiterMiddleware(ipBlockInfoMap, mylimiter.SimpleLimiterOpts{
		Window:        time.Minute,
		MaxReqs:       5,
		BlockDuration: time.Minute,
		Message:       "You have been creating too many rooms. Wait one minute.",
	}), helpers.AuthMiddleware, controllers.HandleCreateRoom)
}
