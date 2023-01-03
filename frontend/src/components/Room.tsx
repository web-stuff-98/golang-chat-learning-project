import { useAuth } from "../context/AuthContext";
import { IRoom, useRooms } from "../context/RoomsContext";
import { useState, useEffect, useRef } from "react";
import classes from "../styles/pages/RoomList.module.scss";
import { useNavigate } from "react-router-dom";
import IconBtn from "./IconBtn";
import { AiFillEdit, AiFillDelete } from "react-icons/ai";
import { IoEnter } from "react-icons/io5";
import { useModal } from "../context/ModalContext";
import { getRoomImage } from "../services/rooms";
import axios, { CancelToken, CancelTokenSource } from "axios";

export default function Room({ room }: { room: IRoom }) {
  const { user } = useAuth();
  const { deleteRoom, updateRoomData, getRoomData } = useRooms();
  const { openModal, closeModal } = useModal();
  const navigate = useNavigate();

  const [fetching, setFetching] = useState(false);

  const imgCancelToken = useRef<CancelToken>();
  const imgCancelSource = useRef<CancelTokenSource>();

  const containerRef = useRef(null);

  const observer = new IntersectionObserver(([entry]) => {
    if (!room) return;
    if (entry.isIntersecting) {
      imgCancelSource.current = axios.CancelToken.source();
      imgCancelToken.current = imgCancelSource.current.token;
      const r = getRoomData(room.ID);
      if (!fetching && !r?.img_url) {
        setFetching(true);
        getRoomImage(room.ID, imgCancelToken.current)
          .then((url) => {
            updateRoomData({ ID: room.ID, img_url: url }, true);
          })
          .catch((e) => {
            if (!axios.isCancel(e)) console.error(e);
          })
          .finally(() => setFetching(false));
      }
    } else {
      setFetching(false);
      if (imgCancelToken.current) {
        imgCancelSource.current?.cancel("Post no longer visible");
      }
    }
  });
  useEffect(() => {
    observer.observe(containerRef.current!);
    return () => {
      observer.disconnect();
      if (imgCancelToken.current) {
        imgCancelSource.current?.cancel("Post no longer visible");
      }
      const r = getRoomData(room.ID);
      if (r) URL.revokeObjectURL(r?.img_url!);
    };
  }, []);

  return (
    <div
      ref={containerRef}
      key={room.ID}
      style={
        room.img_url || room.img_blur
          ? { backgroundImage: `url(${room.img_url || room.img_blur})` }
          : {}
      }
      className={classes.room}
    >
      <h3
        style={
          room.img_url || room.img_blur
            ? {
                color: "white",
                filter: "drop-shadow(0px 2px 3px black)",
              }
            : {}
        }
      >
        {room.name}
      </h3>
      <div
        style={
          room.img_url || room.img_blur
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
            style={room.img_url || room.img_blur ? { color: "white" } : {}}
          />
        )}
        {room.author_id === user?.ID! && (
          <IconBtn
            onClick={() =>
              openModal("Confirm", {
                err: false,
                pen: false,
                msg: "Are you sure you want to delete this room?",
                confirmationCallback: async () => {
                  try {
                    await deleteRoom(room.ID);
                    closeModal();
                  } catch (e) {
                    openModal("Message", {
                      err: true,
                      pen: false,
                      msg: `${e}`,
                    });
                  }
                },
                cancellationCallback: () => {},
              })
            }
            Icon={AiFillDelete}
            name="Delete room"
            ariaLabel="Delete room"
            style={room.img_url || room.img_blur ? { color: "white" } : {}}
          />
        )}
        <IconBtn
          onClick={() => navigate(`/room/${room.ID}`)}
          Icon={IoEnter}
          name="Join room"
          ariaLabel="Join room"
          style={room.img_url || room.img_blur ? { color: "white" } : {}}
        />
      </div>
    </div>
  );
}
