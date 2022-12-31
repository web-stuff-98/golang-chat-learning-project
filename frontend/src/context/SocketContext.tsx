import { useState, useContext, createContext, useEffect } from "react";
import type { ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { useRooms } from "./RoomsContext";
import { useUsers } from "./UsersContext";

/*
  onmessage event_type
  (if there's no event_type that means its a chatroom message)

  chatroom_update
  pfp_update
  chatroom_delete
  user_delete
*/

const SocketContext = createContext<{
  socket?: WebSocket;
}>({
  socket: undefined,
});

export const SocketProvider = ({ children }: { children: ReactNode }) => {
  const { user } = useAuth();
  const { updateUserData, deleteUser } = useUsers();
  const { updateRoomData, deleteRoom, ownRooms, deleteRoomsByAuthor } =
    useRooms();
  const [socket, setSocket] = useState<WebSocket>();

  useEffect(() => {
    if (!user) return;
    //wss <- secure socket protocol.
    const socket = new WebSocket(
      process.env.NODE_ENV === "development"
        ? "ws://localhost:8080/ws/conn"
        : "wss://golang-chat-learning-project.herokuapp.com/ws/conn"
    );
    setSocket(socket);
    socket.onmessage = (e) => {
      const data = JSON.parse(e.data);
      console.log(JSON.stringify(data));
      /* if no event_type field, then its a chatroom message
      socket.onmessage func will be taken over by Room page component
      when the user is in a room. */
      if (!data.event_type) {
        return; 
      }
      if (data.event_type === "chatroom_update") {
        console.log("UPDATE");
        if (!ownRooms) {
          updateRoomData(data);
        } else {
          if (data.author_id === user.ID) {
            updateRoomData(data);
          }
        }
      }
      if (data.event_type === "pfp_update") {
        updateUserData(data);
      }
      if (data.event_type === "user_delete") {
        deleteUser(data.ID);
        deleteRoomsByAuthor(data.ID);
      }
      if (data.event_type === "chatroom_delete") {
        deleteRoom(data.ID);
      }
    };

    return () => {
      socket.close();
    };
  }, [user]);

  return (
    <SocketContext.Provider value={{ socket }}>
      {children}
    </SocketContext.Provider>
  );
};

export const useSocket = () => useContext(SocketContext);
