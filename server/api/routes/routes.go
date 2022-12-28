package routes

import (
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, chatServer *controllers.ChatServer, closeWsChan chan string) {
	app.Post("/api/login", controllers.Login)
	app.Post("/api/welcome", controllers.Welcome)
	app.Post("/api/register", controllers.Register)
	app.Post("/api/updatepfp", controllers.UpdatePfp(chatServer))
	app.Post("/api/refresh", controllers.Refresh(closeWsChan))
	app.Post("/api/logout", controllers.Logout(closeWsChan))
	app.Get("/api/user/:id", controllers.GetUser)

	app.Use("/ws", controllers.HandleWsUpgrade)
	app.Get("/ws/conn", controllers.HandleWsConn(chatServer, closeWsChan))

	app.Get("/api/room/:id", controllers.GetRoom)
	app.Patch("/api/room/:id", controllers.UpdateRoom)
	app.Delete("/api/room/:id", controllers.DeleteRoom(chatServer))
	app.Post("/api/room/:id/image", controllers.UploadRoomImage(chatServer))
	app.Post("/api/room/:id/join", controllers.JoinRoom(chatServer))
	app.Post("/api/room/:id/leave", controllers.LeaveRoom(chatServer))
	app.Get("/api/rooms", controllers.GetRooms)
	app.Post("/api/rooms", controllers.CreateRoom)
}
