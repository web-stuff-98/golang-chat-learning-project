import { useState, useContext, createContext, useEffect } from "react";
import type { ReactNode } from "react";
import { useAuth } from "./AuthContext";

const SocketContext = createContext<{
  socket?: WebSocket;
}>({
  socket: undefined,
});

export const SocketProvider = ({ children }: { children: ReactNode }) => {
  const { user } = useAuth();
  const [socket, setSocket] = useState<WebSocket>();

  useEffect(() => {
    if (!user) return;
    //wss <- secure socket protocol. using this protocol sends the cookie. except I
    //dont know how to configure go-fiber websocket to use this protocol... I have
    //set up a different method for authenticating the socket connection...
    const socket = new WebSocket("ws://localhost:8080/ws/conn");
    setSocket(socket);
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
