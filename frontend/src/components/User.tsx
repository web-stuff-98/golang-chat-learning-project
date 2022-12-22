import { IUser } from "../context/AuthContext";
import classes from "../styles/components/User.module.scss";

import { AiOutlineUser } from "react-icons/ai";

export default function User({
  user = { username: "Username", ID: "123" },
  onClick = () => {},
}: {
  user?: IUser;
  onClick?: () => void;
}) {
  return (
    <div className={classes.container}>
      <span
        style={
          user.base64pfp ? { backgroundImage: `url(${user.base64pfp})` } : {}
        }
        onClick={() => onClick()}
        className={classes.pfp}
      >
        {!user.base64pfp && <AiOutlineUser className={classes.pfpIcon} />}
      </span>
      {user.username}
    </div>
  );
}
