import { useSearchParams } from "react-router-dom";
import classes from "../styles/pages/RoomList.module.scss";

import { useEffect, useState } from "react";

import { getRooms } from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";

export default function RoomList() {
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
      {rooms.map((room: any) => room.name)}
      <ResMsg resMsg={resMsg} />
    </div>
  );
}
