import { useState, useContext, createContext } from "react";
import type { ReactNode } from "react";
import { useSocket } from "./SocketContext";

const ChatContext = createContext<{
  sendMessage: (msg: string) => void;
  joinRoom: (id: string) => void;
  leaveRoom: (id: string) => void;
}>({
  sendMessage: () => {},
  joinRoom: () => {},
  leaveRoom: () => {},
});

export const ChatProvider = ({ children }: { children: ReactNode }) => {
  const socket = useSocket();

  const sendMessage = (msg: string) => {

  };

  const joinRoom = (id: string) => {

  };

  const leaveRoom = (id: string) => {
    
  };

  return (
    <ChatContext.Provider value={{ sendMessage, joinRoom, leaveRoom }}>
      {children}
    </ChatContext.Provider>
  );
};
