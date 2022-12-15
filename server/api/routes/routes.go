package routes

import (
	"cookie-session/api/controllers"

	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, chatServer *controllers.ChatServer, closeWsChan chan string) {

	//HTTP api routes not related to chat
	app.Post("/api/login", controllers.Login)
	app.Post("/api/welcome", controllers.Welcome)
	app.Post("/api/register", controllers.Register)
	app.Post("/api/refresh", controllers.Refresh(closeWsChan))
	app.Post("/api/logout", controllers.Logout(closeWsChan))

	//WS api routes and chat related HTTP api routes
	app.Use("/ws", controllers.HandleWsUpgrade)
	app.Get("/ws/conn", controllers.HandleWsConn(chatServer, closeWsChan))
	app.Get("/api/room/:id", controllers.GetRoom)
	app.Get("/api/rooms", controllers.GetRooms)
	app.Post("/api/rooms", controllers.CreateRoom)
}
