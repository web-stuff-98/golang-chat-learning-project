import classes from "../styles/pages/Room.module.scss";

import { useEffect, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useSocket } from "../context/SocketContext";
import { getRoom, joinRoom, leaveRoom } from "../services/rooms";
import { useNavigate, useParams } from "react-router-dom";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { useAuth } from "../context/AuthContext";

export interface IMsg {
  content: string;
  uid: string;
  timestamp: Date;
}

export default function Room() {
  const { socket } = useSocket();
  const { id } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate()

  const [joined, setJoined] = useState(false);
  const [messageInput, setMessageInput] = useState("");
  const [messages, setMessages] = useState<IMsg[]>([]);
  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    socket?.send(messageInput);
    setMessages((p) => [
      ...p,
      { content: messageInput, timestamp: new Date(), uid: user?.ID! },
    ]);
  };

  useEffect(() => {
    if (!joined) {
      joinRoom(id as string)
        .then((room) => {
          setMessages(room.messages)
          setJoined(true);
        })
        .catch((e) => setResMsg({ msg: `${e}`, err: true, pen: false }));
    }
    return () => {
      leaveRoom(id as string).catch((e) =>
        setResMsg({ msg: `${e}`, err: true, pen: false })
      );
    };
  }, [id]);

  useEffect(() => {
    if (!socket) return;
    socket.onmessage = (e) => {
      setMessages((old) => [...old, e.data]);
      console.log("Received message : " + e.data);
    };
  }, [socket]);

  return (
    <div className={classes.container}>
      <div className={classes.messages}>
        {messages.map((msg) => (
          <div className={classes.message}>
            <div className={classes.messageContent}>{msg.content}</div>
          </div>
        ))}
      </div>
      <form onSubmit={handleSubmit}>
        <input
          onChange={(e: ChangeEvent<HTMLInputElement>) =>
            setMessageInput(e.target.value)
          }
          type="text"
        />
        <button type="submit">Send</button>
      </form>
      <button className={classes.backButton} onClick={() => navigate("/room/list")} type="button">Back</button>
      <ResMsg resMsg={resMsg} />
    </div>
  );
}
