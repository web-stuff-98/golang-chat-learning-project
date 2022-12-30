import { useAuth } from "../context/AuthContext";
import { IRoom, useRooms } from "../context/RoomsContext";
import { useState, useEffect, useRef, useLayoutEffect } from "react";
import classes from "../styles/pages/RoomList.module.scss";
import { useNavigate } from "react-router-dom";
import IconBtn from "./IconBtn";
import { AiFillEdit, AiFillDelete } from "react-icons/ai";
import { IoEnter } from "react-icons/io5";
import { useModal } from "../context/ModalContext";
import { getRoomImage } from "../services/rooms";
import { useSocket } from "../context/SocketContext";

export default function Room({ room }: { room: IRoom }) {
  const { user } = useAuth();
  const { deleteRoom, updateRoomData, getRoomData } = useRooms();
  const { openModal, closeModal } = useModal();
  const { socket } = useSocket();
  const navigate = useNavigate();

  const [fetching, setFetching] = useState(false);

  const containerRef = useRef(null);

  const observer = new IntersectionObserver(([entry]) => {
    if (!room) return;
    if (entry.isIntersecting) {
      setFetching(true);
      if (!fetching)
        getRoomImage(room.ID)
          .then((url) => {
            updateRoomData({ID:room.ID, img_url: url})
          })
          .catch((e) => {
            console.error(e);
          })
          .finally(() => setFetching(false));
    } else {
      const r = getRoomData(room.ID)
      URL.revokeObjectURL(r?.img_url!);
      updateRoomData({ID:room.ID, img_url:undefined})
      setFetching(false);
    }
  });
  useLayoutEffect(() => {
    observer.observe(containerRef.current!);
    return () => {
      observer.disconnect();
    };
    //putting the ref in the dependency array was the only way to get this working properly for some reason
  }, [containerRef.current]);

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
