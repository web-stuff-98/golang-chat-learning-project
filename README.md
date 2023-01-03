# Go-Chat

## This is my first golang project, its a chat app that uses MongoDB and Websockets.

Features :

- Refresh tokens
- Customizable profiles
- Customizable rooms
- Custom Rate limiter
- Group chat
- Customizable profiles
- Automatic account deletion
- Automatic clean up via changestream

It doesn't work very well, for some reason sometimes messages wont be received after leaving
a room and joining another one. Some API routes will time out or just crash the server, and
logging out creates a new cookie with the same name instead of removing the old one.
https://golang-chat-learning-project.herokuapp.com


I followed along with this medium article to deploy it
https://medium.com/jaysonmulwa/deploying-a-go-fiber-go-react-app-to-heroku-using-docker-7379ed47e0fc