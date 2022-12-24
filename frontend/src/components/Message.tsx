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
      style={reverse ? { flexDirection: "row-reverse" } : {}}
      className={classes.message}
    >
      <User
        uid={msg.uid}
        date={new Date(msg.timestamp)}
        reverse={reverse}
        user={getUserData(msg.uid)}
      />
      <div className={classes.messageContent}>{msg.content}</div>
    </div>
  );
}
