package routes

import (
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, chatServer *controllers.ChatServer, closeWsChan chan string) {
	app.Post("/api/login", controllers.Login)
	app.Post("/api/welcome", controllers.Welcome)
	app.Post("/api/register", controllers.Register)
	app.Post("/api/updatepfp", helpers.AuthMiddleware, controllers.UpdatePfp(chatServer))
	app.Post("/api/deleteacc", helpers.AuthMiddleware, controllers.DeleteUser)
	app.Post("/api/refresh", controllers.Refresh(closeWsChan))
	app.Post("/api/logout", controllers.Logout(closeWsChan))
	app.Get("/api/user/:id", helpers.AuthMiddleware, controllers.GetUser)

	app.Use("/ws", controllers.HandleWsUpgrade)
	app.Get("/ws/conn", controllers.HandleWsConn(chatServer, closeWsChan))

	app.Get("/api/room/:id", helpers.AuthMiddleware, controllers.GetRoom)
	app.Patch("/api/room/:id", helpers.AuthMiddleware, controllers.UpdateRoom)
	app.Delete("/api/room/:id", helpers.AuthMiddleware, controllers.DeleteRoom(chatServer))
	app.Post("/api/room/:id/image", helpers.AuthMiddleware, controllers.UploadRoomImage(chatServer))
	app.Post("/api/room/:id/join", helpers.AuthMiddleware, controllers.JoinRoom(chatServer))
	app.Post("/api/room/:id/leave", helpers.AuthMiddleware, controllers.LeaveRoom(chatServer))
	app.Get("/api/rooms", helpers.AuthMiddleware, controllers.GetRooms)
	app.Post("/api/rooms", helpers.AuthMiddleware, controllers.CreateRoom)
}
