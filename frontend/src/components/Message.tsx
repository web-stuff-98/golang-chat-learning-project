import { useUsers } from "../context/UsersContext";
import { IMsg } from "../routes/Room";
import classes from "../styles/pages/Room.module.scss";
import User from "./User";

export default function Message({
  msg,
  reverse,
}: {
  msg: IMsg;
  reverse: boolean;
}) {
  const { getUserData } = useUsers();
  return (
    <div
      style={reverse ? { alignItems: "flex-end" } : {}}
      className={classes.message}
    >
      <div
        className={classes.userAndTextContent}
        style={reverse ? { flexDirection: "row-reverse" } : {}}
      >
        <User
          uid={msg.uid}
          date={new Date(msg.timestamp)}
          reverse={reverse}
          user={getUserData(msg.uid)}
        />
        <div
          style={reverse ? { textAlign: "right" } : {}}
          className={classes.messageContent}
        >
          {msg.content}
        </div>
      </div>
      {msg.has_attachment ? "HAS ATTACHMENT | " : "NO ATTACHMENT | "}
      {msg.attachment_pending ? "ATTACHMENT PENDING" : "ATTACHMENT NOT PENDING"}
      <div className={classes.imageAttachment}></div>
    </div>
  );
}
