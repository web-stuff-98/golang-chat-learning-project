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
        <button
          aria-label="Open all rooms"
          onClick={() => navigate("/room/list")}
        >
          All rooms
        </button>
        <button
          aria-label="Open your rooms"
          onClick={() => navigate("/room/list?own=true")}
        >
          Your rooms
        </button>
        <button
          aria-label="Create a room"
          onClick={() => navigate("/room/edit")}
        >
          Create
        </button>
      </div>
    </ProtectedRoute>
  );
}
