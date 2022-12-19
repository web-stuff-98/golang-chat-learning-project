import { useNavigate, useSearchParams } from "react-router-dom";
import classes from "../styles/pages/RoomList.module.scss";

import { useEffect, useState } from "react";

import { getRooms } from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";

import { IoEnter } from "react-icons/io5";
import IconBtn from "../components/IconBtn";

export default function RoomList() {
  const navigate = useNavigate();

  const [searchParams] = useSearchParams();

  const [rooms, setRooms] = useState([]);
  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const getRoomList = async () => {
    try {
      setResMsg({ msg: "", err: false, pen: true });
      const data = await getRooms(searchParams.get("own") ? true : false);
      setRooms(data ?? []);
      setResMsg({ msg: "", err: false, pen: false });
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
        {rooms.map((room: any) => (
          <div className={classes.room}>
            <h3>{room.name}</h3>
            <IconBtn onClick={() => navigate(`/room/${room.ID}`)} Icon={IoEnter} name="Join room" ariaLabel="Join room" />
          </div>
        ))}
      </div>
      <button className={classes.back} aria-label="Back" onClick={() => navigate("/room/menu")}>
        Back
      </button>
      <ResMsg resMsg={resMsg} />
    </div>
  );
}
