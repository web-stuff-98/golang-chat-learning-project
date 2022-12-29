import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import classes from "../styles/pages/RoomMenu.module.scss";
import ProtectedRoute from "./ProtectedRoute";

export default function RoomMenu() {
  const navigate = useNavigate();
  const { user } = useAuth();

  return (
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        {<button onClick={() => navigate("/room/list")}>All rooms</button>}
        {
          <button onClick={() => navigate("/room/list?own=true")}>
            Your rooms
          </button>
        }
        {<button onClick={() => navigate("/room/edit")}>Create</button>}
      </div>
    </ProtectedRoute>
  );
}
