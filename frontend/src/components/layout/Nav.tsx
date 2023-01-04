import classes from "./Layout.module.scss";

import { Link } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";

import User from "../User";
import { useState } from "react";
import { useInterface } from "../../context/InterfaceContext";

import { MdMenu } from "react-icons/md";

export default function Nav() {
  const { user, logout } = useAuth();
  const { state } = useInterface();
  const [mobileOpen, setMobileOpen] = useState(false);

  const isMobile = () => state.dimensions.width <= 475;

  return (
    <nav
      style={
        isMobile()
          ? {
              justifyContent: "flex-end",
              alignItems: "center",
              ...(mobileOpen
                ? {
                    height: "12.5rem",
                    justifyContent: "space-between",
                    alignItems: "flex-end",
                    paddingLeft: "3px",
                  }
                : {}),
            }
          : {}
      }
    >
      {isMobile() && (
        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          aria-label={mobileOpen ? "Close nav links" : "Open nav links"}
          className={classes.menuIcon}
        >
          <MdMenu />
        </button>
      )}
      {(!isMobile() || mobileOpen) && (
        <div
          style={
            mobileOpen && isMobile()
              ? {
                  flexDirection: "column",
                  alignItems: "flex-start",
                  width: "fit-content",
                  marginBottom: "calc(var(--nav-height) + var(--padding))",
                }
              : {}
          }
          className={classes.navLinks}
        >
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
                style={{
                  background: "none",
                  border: "none",
                  padding: "none",
                }}
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
      )}
      {user && (
        <div className={classes.userContainer}>
          <User light reverse uid={user?.ID} user={user} />
        </div>
      )}
    </nav>
  );
}
