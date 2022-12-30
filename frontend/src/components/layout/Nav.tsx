import classes from "./Layout.module.scss";

import { Link } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";

import User from "../User";

export default function Nav() {
  const { user, logout } = useAuth();

  return (
    <nav>
      <div className={classes.navLinks}>
        <Link to="/">
          <span>Home</span>
        </Link>
        {!user && (
          <>
            <Link to="/login">
              <span>Login</span>
            </Link>
            <Link to="/register">
              <span>Register</span>
            </Link>
          </>
        )}
        <Link to="/policy">
          <span>Policy</span>
        </Link>
        {user && (
          <>
            <button
              onClick={() => logout()}
              aria-label="Logout"
              style={{ background: "none", border: "none", padding: "none" }}
            >
              <span>Logout</span>
            </button>
            <Link to="/room/menu">
              <span>Rooms</span>
            </Link>
            <Link to="/settings">
              <span>Settings</span>
            </Link>
          </>
        )}
      </div>
      {user && (
        <div className={classes.userContainer}>
          <User light reverse uid={user?.ID} user={user} />
        </div>
      )}
    </nav>
  );
}
