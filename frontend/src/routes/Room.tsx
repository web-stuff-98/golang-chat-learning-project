import classes from "../styles/pages/Room.module.scss";
import { useEffect, useState, useRef } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useSocket } from "../context/SocketContext";
import { joinRoom, leaveRoom } from "../services/rooms";
import { useNavigate, useParams } from "react-router-dom";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { useAuth } from "../context/AuthContext";
import { useUsers } from "../context/UsersContext";
import Message from "../components/Message";
import { IRoom, useRooms } from "../context/RoomsContext";
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
  const { cacheUserData, updateUserData } = useUsers();
  const navigate = useNavigate();

  const msgsBottomRef = useRef<HTMLSpanElement>(null);
  const [joined, setJoined] = useState(false);
  const [messageInput, setMessageInput] = useState("");
  const [messages, setMessages] = useState<IMsg[]>([]);
  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  useEffect(() => {
    if (!messages) return;
    msgsBottomRef.current?.scrollIntoView({ behavior: "auto" });
  }, [messages]);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    socket?.send(messageInput);
    setMessageInput("");
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
      console.log(JSON.stringify(data));
      if (!data.event_type) {
        //if no event_type, then its a normal room message, so don't ignore it
        cacheUserData(data.uid);
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
        if (data.event_type === "pfp_update") {
          updateUserData(data)
        }
      }
    };
  }, [socket]);

  const renderRoomName = (room?: IRoom) => (room ? room.name : "");

  return (
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        <div className={classes.roomName}>
          {renderRoomName(getRoomData(id as string))}
        </div>
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
          <span aria-hidden={true} ref={msgsBottomRef} />
        </div>
        <form onSubmit={handleSubmit}>
          <input
            value={messageInput}
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
