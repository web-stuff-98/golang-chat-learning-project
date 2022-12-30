import { useNavigate, useSearchParams } from "react-router-dom";
import classes from "../styles/pages/RoomList.module.scss";
import { useEffect, useState } from "react";
import { getRooms } from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { IRoom, useRooms } from "../context/RoomsContext";
import { useAuth } from "../context/AuthContext";
import ProtectedRoute from "./ProtectedRoute";
import Room from "../components/Room";
import { BsChevronLeft, BsChevronRight } from "react-icons/bs";

export default function RoomList() {
  const navigate = useNavigate();
  const { setAllRooms, rooms, setOwnRooms } = useRooms();
  const { user } = useAuth();

  const [searchParams] = useSearchParams();

  const [pageIndex, setPageIndex] = useState(0);

  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const getRoomList = async () => {
    try {
      setPageIndex(0);
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

  const nextPage = () => {
    setPageIndex((i) => Math.min(i + 1, Math.ceil(rooms.length / 20) - 1));
  };

  const prevPage = () => {
    setPageIndex((i) => Math.max(i - 1, 0));
  };

  return (
    <ProtectedRoute user={user}>
      <div className={classes.container}>
        <div className={classes.rooms}>
          <ResMsg resMsg={resMsg} />
          {!resMsg.pen &&
            rooms
              .slice(pageIndex * 20, pageIndex * 20 + 20)
              .map((room: IRoom) => <Room key={room.ID} room={room} />)}
        </div>
        <div className={classes.paginationControls}>
          <button onClick={() => prevPage()} aria-label="Previous page">
            <BsChevronLeft />
          </button>
          <div className={classes.pageNumber}>{pageIndex + 1}</div>/
          <div className={classes.maxPage}>{Math.ceil(rooms.length / 20)}</div>
          <button onClick={() => nextPage()} aria-label="Next page">
            <BsChevronRight />
          </button>
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
