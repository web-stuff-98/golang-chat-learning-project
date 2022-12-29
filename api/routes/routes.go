package routes

import (
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, chatServer *controllers.ChatServer, closeWsChan chan string) {
	app.Post("/api/welcome", controllers.Welcome)
	app.Post("/api/user/login", controllers.HandleLogin)
	app.Post("/api/user/register", controllers.HandleRegister)
	app.Post("/api/user/updatepfp", helpers.AuthMiddleware, controllers.HandleUpdatePfp(chatServer))
	app.Post("/api/user/deleteacc", helpers.AuthMiddleware, controllers.HandleDeleteUser)
	app.Post("/api/user/refresh", controllers.HandleRefresh(closeWsChan))
	app.Post("/api/user/logout", controllers.HandleLogout(closeWsChan))
	app.Get("/api/user/:id", helpers.AuthMiddleware, controllers.HandleGetUser)

	app.Use("/ws", controllers.HandleWsUpgrade)
	app.Get("/ws/conn", controllers.HandleWsConn(chatServer, closeWsChan))

	app.Get("/api/room/:id", helpers.AuthMiddleware, controllers.HandleGetRoom)
	app.Patch("/api/room/:id", helpers.AuthMiddleware, controllers.HandleUpdateRoom)
	app.Delete("/api/room/:id", helpers.AuthMiddleware, controllers.HandleDeleteRoom(chatServer))
	app.Post("/api/room/:id/image", helpers.AuthMiddleware, controllers.HandleUploadRoomImage(chatServer))
	app.Post("/api/room/:id/join", helpers.AuthMiddleware, controllers.HandleJoinRoom(chatServer))
	app.Post("/api/room/:id/leave", helpers.AuthMiddleware, controllers.HandleLeaveRoom(chatServer))
	app.Get("/api/room/rooms", helpers.AuthMiddleware, controllers.HandleGetRooms)
	app.Post("/api/room", helpers.AuthMiddleware, controllers.HandleCreateRoom)
}
