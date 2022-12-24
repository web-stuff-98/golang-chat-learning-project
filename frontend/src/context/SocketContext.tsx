import { useState, useContext, createContext, useEffect } from "react";
import type { ReactNode } from "react";
import { useAuth } from "./AuthContext";
import { useRooms } from "./RoomsContext";
import { useUsers } from "./UsersContext";

const SocketContext = createContext<{
  socket?: WebSocket;
}>({
  socket: undefined,
});

export const SocketProvider = ({ children }: { children: ReactNode }) => {
  const { user } = useAuth();
  const { updateUserData } = useUsers();
  const { updateRoomData, ownRooms } = useRooms();
  const [socket, setSocket] = useState<WebSocket>();

  useEffect(() => {
    if (!user) return;
    //wss <- secure socket protocol. using this protocol sends the cookie. except I
    //dont know how to configure go-fiber websocket to use this protocol... I have
    //set up a different method for authenticating the socket connection...
    const socket = new WebSocket("ws://localhost:8080/ws/conn");
    setSocket(socket);
    socket.onmessage = (e) => {
      let data = e.data;
      delete data.event_type;
      if (!Object.keys(e.data).includes("event_type")) return; //if no event_type field, then its a chatroom message
      if (e.data.event_type === "chatroom_update") {
        if (!ownRooms) {
          updateRoomData(data);
        } else if (e.data.author_id === user.ID) {
          updateRoomData(data);
        }
      }
      if (e.data.event_type === "pfp_update") {
        updateUserData(data);
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
