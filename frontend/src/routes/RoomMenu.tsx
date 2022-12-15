import { useNavigate } from "react-router-dom";
import classes from "../styles/pages/RoomMenu.module.scss";

export default function RoomMenu() {
  const navigate = useNavigate();

  return (
    <div className={classes.container}>
      {<button onClick={() => navigate("/room/list")}>All rooms</button>}
      {<button onClick={() => navigate("/room/list?own=true")}>Your rooms</button>}
      {<button onClick={() => navigate("/room/edit")}>Create room</button>}
    </div>
  );
}
