import { useNavigate, useSearchParams } from "react-router-dom";
import classes from "../styles/pages/RoomList.module.scss";

import { useEffect, useState } from "react";

import { getRooms } from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";

import { IoEnter } from "react-icons/io5";
import { AiFillEdit, AiFillDelete } from "react-icons/ai";
import IconBtn from "../components/IconBtn";
import { IRoom, useRooms } from "../context/RoomsContext";
import { useAuth } from "../context/AuthContext";

export default function RoomList() {
  const navigate = useNavigate();
  const { setAllRooms, rooms, setOwnRooms, deleteRoom } = useRooms();
  const { user } = useAuth();

  const [searchParams] = useSearchParams();

  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const getRoomList = async () => {
    try {
      setResMsg({ msg: "Loading rooms...", err: false, pen: true });
      const data = await getRooms(searchParams.get("own") ? true : false);
      setAllRooms(data ?? []);
      setOwnRooms(searchParams.get("own") ? true : false);
      setResMsg({
        msg: data ? "" : "You have no rooms",
        err: false,
        pen: false,
      });
    } catch (e) {
      setResMsg({ msg: `${e}`, err: true, pen: false });
    }
  };

  useEffect(() => {
    getRoomList();
  }, [searchParams]);

  return (
    <div className={classes.container}>
      <div className={classes.rooms}>
        <ResMsg resMsg={resMsg} />
        {rooms.map((room: IRoom) => (
          <div
            key={room.ID}
            style={
              room.base64image
                ? { backgroundImage: `url(${room.base64image})` }
                : {}
            }
            className={classes.room}
          >
            <h3
              style={
                room.base64image
                  ? { color: "white", filter: "drop-shadow(0px 2px 3px black)" }
                  : {}
              }
            >
              {room.name}
            </h3>
            <div
              style={
                room.base64image
                  ? { filter: "drop-shadow(0px 2px 3px black)" }
                  : {}
              }
              className={classes.icons}
            >
              {room.author_id === user?.ID! && (
                <IconBtn
                  onClick={() => navigate(`/room/edit/${room.ID}`)}
                  Icon={AiFillEdit}
                  name="Edit room"
                  ariaLabel="Edit room"
                  style={room.base64image ? { color: "white" } : {}}
                />
              )}
              {room.author_id === user?.ID! && (
                <IconBtn
                  onClick={() => deleteRoom(room.ID)}
                  Icon={AiFillDelete}
                  name="Delete room"
                  ariaLabel="Delete room"
                  style={room.base64image ? { color: "white" } : {}}
                />
              )}
              <IconBtn
                onClick={() => navigate(`/room/${room.ID}`)}
                Icon={IoEnter}
                name="Join room"
                ariaLabel="Join room"
                style={room.base64image ? { color: "white" } : {}}
              />
            </div>
          </div>
        ))}
      </div>
      <button
        className={classes.back}
        aria-label="Back"
        onClick={() => navigate("/room/menu")}
      >
        Back
      </button>
    </div>
  );
}
