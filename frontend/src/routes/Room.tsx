import classes from "../styles/pages/Room.module.scss";
import { useEffect, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useSocket } from "../context/SocketContext";
import { joinRoom, leaveRoom } from "../services/rooms";
import { useNavigate, useParams } from "react-router-dom";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { useAuth } from "../context/AuthContext";
import { useUsers } from "../context/UsersContext";
import Message from "../components/Message";
import { useRooms } from "../context/RoomsContext";
import ProtectedRoute from "./ProtectedRoute";

export interface IMsg {
  content: string;
  uid: string;
  timestamp: Date;
}

export default function Room() {
  const { socket } = useSocket();
  const { id } = useParams();
  const { user } = useAuth();
  const { getRoomData } = useRooms();
  const { cacheUserData } = useUsers();
  const navigate = useNavigate();

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
          setMessages(room.messages);
          setJoined(true);
          room.messages.forEach((msg: IMsg) => cacheUserData(msg.uid));
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
      let data = JSON.parse(e.data);
      if (!data.event_type) {
        //if no event_type, then its a normal room message, so don't ignore it
        delete data.event_type;
        cacheUserData(e.data.uid);
        setMessages((old) => [
          ...old,
          { ...data, timestamp: new Date().toISOString() },
        ]);
      } else {
        if (data.event_type === "user_delete") {
          const r = getRoomData(id as string);
          setMessages((old) => [...old.filter((msg) => msg.uid !== data.ID)]);
          if (r) {
            if (r.author_id === data.ID) {
              navigate("/room/list");
            }
          }
        }
        if (data.event_type === "chatroom_delete") {
          if (data.ID === id) {
            navigate("/room/list");
          }
        }
      }
    };
  }, [socket]);

  return (
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        <div className={classes.messages}>
          {messages && messages.length && !resMsg.pen ? (
            messages.map((msg) => (
              <Message msg={msg} reverse={msg.uid !== user?.ID} />
            ))
          ) : (
            <p className={classes.roomHasNoMessages}>
              This room has received no messages.
            </p>
          )}
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
        <button
          className={classes.backButton}
          onClick={() => navigate("/room/list")}
          type="button"
        >
          Back
        </button>
        <ResMsg resMsg={resMsg} />
      </div>
    </ProtectedRoute>
  );
}
