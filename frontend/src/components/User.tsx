import { useRef, useLayoutEffect } from "react";

import { IUser } from "../context/AuthContext";
import classes from "../styles/components/User.module.scss";

import { AiOutlineUser } from "react-icons/ai";
import { useUsers } from "../context/UsersContext";

export default function User({
  user = { username: "Username", ID: "123" },
  uid = "",
  onClick = undefined,
}: {
  user?: IUser;
  uid: string;
  onClick?: () => void;
}) {
  const { userEnteredView, cacheUserData, userLeftView } = useUsers();
  const containerRef = useRef<HTMLDivElement>(null);

  const observer = new IntersectionObserver(([entry]) => {
    if (!uid || uid === "undefined") return;
    if (entry.isIntersecting) {
      userEnteredView(uid);
      cacheUserData(uid);
    } else {
      userLeftView(uid);
    }
  });
  useLayoutEffect(() => {
    observer.observe(containerRef.current!);
    return () => {
      if (uid) userLeftView(uid);
      observer.disconnect();
    };
  }, [containerRef.current]);

  return (
    <div ref={containerRef} className={classes.container}>
      {user && (
        <>
          <span
            style={{
              ...(user.base64pfp
                ? { backgroundImage: `url(${user.base64pfp})` }
                : {}),
              ...(onClick ? { cursor: "pointer" } : {}),
            }}
            onClick={() => {
              if (onClick) onClick();
            }}
            className={classes.pfp}
          >
            {!user.base64pfp && <AiOutlineUser className={classes.pfpIcon} />}
          </span>
          {user.username}
        </>
      )}
    </div>
  );
}
