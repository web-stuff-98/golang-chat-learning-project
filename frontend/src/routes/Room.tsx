import classes from "../styles/pages/Room.module.scss";
import { useEffect, useState, useRef, useCallback } from "react";
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
import { useModal } from "../context/ModalContext";
import IconBtn from "../components/IconBtn";
import { IoSend } from "react-icons/io5";
import { AiFillFile } from "react-icons/ai";

export interface IMsg {
  content: string;
  uid: string;
  timestamp: Date;
  ID?: string;
}

export default function Room() {
  const { socket } = useSocket();
  const { id } = useParams();
  const { user } = useAuth();
  const { getRoomData } = useRooms();
  const { cacheUserData, updateUserData } = useUsers();
  const { openModal } = useModal();
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
    socket?.send(
      JSON.stringify({ content: messageInput, has_attachment: false })
    );
    setMessageInput("");
    setMessages((p) => [
      ...p,
      {
        content: messageInput,
        timestamp: new Date(),
        uid: user?.ID!,
        ID: `${Math.random()}${Math.random()}`,
      },
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

  const messageListener = useCallback((e: any) => {
    let data = JSON.parse(e.data);
    if (!data.event_type) {
      //if no event_type, then its a normal room message from another user
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
        updateUserData(data);
      }
      if (data.event_type === "chatroom_err") {
        openModal("Message", {
          msg: e.data.content,
          err: true,
          pen: false,
        });
      }
    }
  }, []);

  useEffect(() => {
    if (!socket) return;
    socket.addEventListener("message", messageListener);
    return () => {
      socket.removeEventListener("message", messageListener);
    };
  }, [socket]);

  const renderRoomName = (room?: IRoom) => (room ? room.name : "");

  const msgFormRef = useRef<HTMLFormElement>(null);
  return (
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        <div className={classes.roomName}>
          {renderRoomName(getRoomData(id as string))}
        </div>
        <div className={classes.messages}>
          {messages && messages.length && !resMsg.pen ? (
            messages.map((msg) => (
              <Message key={msg.ID} msg={msg} reverse={msg.uid !== user?.ID} />
            ))
          ) : (
            <p className={classes.roomHasNoMessages}>
              This room has received no messages.
            </p>
          )}
          <span aria-hidden={true} ref={msgsBottomRef} />
        </div>
        <form ref={msgFormRef} onSubmit={handleSubmit}>
          <IconBtn
            name="Select attachment"
            ariaLabel="Select attachment"
            Icon={AiFillFile}
          />
          <input
            value={messageInput}
            onChange={(e: ChangeEvent<HTMLInputElement>) =>
              setMessageInput(e.target.value)
            }
            type="text"
          />
          <IconBtn type="submit" name="Send" ariaLabel="Send" Icon={IoSend} />
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
