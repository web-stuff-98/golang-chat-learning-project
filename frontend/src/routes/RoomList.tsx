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
import ProtectedRoute from "./ProtectedRoute";
import { useModal } from "../context/ModalContext";
import Room from "../components/Room";

export default function RoomList() {
  const navigate = useNavigate();
  const { openModal, closeModal } = useModal();
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
      const data = await getRooms(searchParams.get("own") ? true : false);
      setResMsg({ msg: "Loading rooms...", err: false, pen: true });
      setAllRooms(data ?? []);
      setOwnRooms(searchParams.get("own") ? true : false);
      setResMsg({
        msg: data
          ? ""
          : searchParams.get("own")
          ? "You have no rooms"
          : "There are no rooms",
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
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        <div className={classes.rooms}>
          <ResMsg resMsg={resMsg} />
          {!resMsg.pen &&
            rooms.map((room: IRoom) => (
              <Room room={room}/>
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
    </ProtectedRoute>
  );
}
