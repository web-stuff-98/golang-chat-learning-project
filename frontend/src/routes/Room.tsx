import classes from "../styles/pages/Room.module.scss";
import { useEffect, useState, useRef, useCallback } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useSocket } from "../context/SocketContext";
import { joinRoom, leaveRoom, uploadAttachment } from "../services/rooms";
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
  has_attachment: boolean;
  attachment_pending: boolean;
  attachment_type?: string;
  attachment_error?: boolean;
}

export default function Room() {
  const { socket } = useSocket();
  const { id } = useParams();
  const { user } = useAuth();
  const { getRoomData } = useRooms();
  const { cacheUserData, updateUserData } = useUsers();
  const { openModal } = useModal();
  const navigate = useNavigate();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const msgsBottomRef = useRef<HTMLSpanElement>(null);
  const [joined, setJoined] = useState(false);
  const [messageInput, setMessageInput] = useState("");
  const [file, setFile] = useState<File>();
  const fileRef = useRef<File>();
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
      JSON.stringify({
        content: messageInput,
        has_attachment: fileRef.current ? true : false,
      })
    );
    setMessageInput("");
  };

  const handleFileInput = (e: ChangeEvent<HTMLInputElement>) => {
    if (!e.target?.files) return;
    if (!e.target.files[0]) return;
    const file = e.target.files[0];
    if (file.size > 20 * 1024 * 1024) {
      openModal("Message", {
        msg: "You cannot select a file larger than 20mb",
        err: true,
        pen: false,
      });
      return;
    }
    setFile(file);
    fileRef.current = file;
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

  const messageListener = useCallback(
    (e: any) => {
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
        if (data.event_type === "attachment_complete") {
          setMessages((old) => {
            let newMsgs = old;
            const i = old.findIndex((m) => m.ID === data.ID);
            if (i === -1) return old;
            newMsgs[i].attachment_pending = false;
            newMsgs[i].attachment_type = data.attachment_type;
            return [...newMsgs];
          });
        }
        if (data.event_type === "message_delete") {
          setMessages((old) => [...old.filter((m) => m.ID !== data.ID)]);
        }
        if (data.event_type === "attachment_upload") {
          uploadAttachment(data.roomID, data.ID, fileRef.current as File)
            .then(() => {
              setFile(undefined);
              fileRef.current = undefined;
            })
            .catch(() => {
              openModal("Message", {
                msg: "Error uploading attachmnet",
                err: true,
                pen: false,
              });
            });
        }
      }
    },
    [socket]
  );

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
          <input
            onChange={handleFileInput}
            type="file"
            style={{ display: "none" }}
            ref={fileInputRef}
          />
          <IconBtn
            name="Select attachment"
            ariaLabel="Select attachment"
            Icon={AiFillFile}
            style={file ? { color: "lime" } : {}}
            onClick={() => fileInputRef.current?.click()}
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
