import classes from "../styles/pages/RoomEditor.module.scss";
import formClasses from "../styles/FormClasses.module.scss";
import { useNavigate, useParams } from "react-router-dom";
import { useRef, useState, useEffect } from "react";
import type { FormEvent, ChangeEvent } from "react";
import {
  createRoom,
  getRoomImage,
  updateRoom,
  uploadRoomImage,
} from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { useRooms } from "../context/RoomsContext";
import ProtectedRoute from "./ProtectedRoute";
import { useAuth } from "../context/AuthContext";
import axios, { CancelToken, CancelTokenSource } from "axios";

export default function RoomEditor() {
  const { id } = useParams();
  const { rooms } = useRooms();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [nameInput, setNameInput] = useState("");
  const [coverImageURL, setCoverImageURL] = useState("");
  const coverImageFileRef = useRef<File>();

  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const imgCancelSource = useRef<CancelTokenSource>();
  const imgCancelToken = useRef<CancelToken>();

  useEffect(() => {
    if (!id) return;
    setNameInput(rooms.find((r) => r.ID === id)?.name!);
    imgCancelSource.current = axios.CancelToken.source();
    imgCancelToken.current = imgCancelSource.current?.token;
    getRoomImage(id, imgCancelToken.current)
      .then((url) => setCoverImageURL(url))
      .catch((e) => {
        if (!axios.isCancel(e)) {
          setCoverImageURL("");
          console.error(e);
        }
      });
    return () => {
      if (imgCancelToken.current) {
        imgCancelSource.current?.cancel("Image no longer visible");
      }
    };
  }, [id]);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      setResMsg({
        msg: id ? "Updating room" : "Creating room",
        err: false,
        pen: true,
      });
      const prom = id
        ? updateRoom(id, {
            name: nameInput,
          })
        : createRoom({
            name: nameInput,
          });
      const data = await prom;
      if (coverImageFileRef.current) {
        await uploadRoomImage(id ?? data.ID, coverImageFileRef.current);
      }
      setResMsg({ msg: "", err: false, pen: false });
    } catch (e) {
      setResMsg({ msg: `${e}`, pen: false, err: true });
    }
  };

  const handleCoverImage = async (e: ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files) return;
    if (!e.target.files[0]) return;
    const file = e.target.files[0];
    try {
      setCoverImageURL(URL.createObjectURL(file));
      coverImageFileRef.current = file;
    } catch (e) {
      setResMsg({ msg: "Image error", err: true, pen: false });
    }
  };

  const fileInputRef = useRef<HTMLInputElement>(null);
  return (
    <ProtectedRoute user={user}>
      <form onSubmit={handleSubmit} className={classes.container}>
        <div className={formClasses.inputLabelWrapper}>
          <label htmlFor="name">Room Name</label>
          <input
            id="name"
            name="name"
            value={nameInput}
            onChange={(e: ChangeEvent<HTMLInputElement>) =>
              setNameInput(e.target.value)
            }
          />
        </div>
        <input
          ref={fileInputRef}
          style={{ display: "none" }}
          onChange={handleCoverImage}
          accept=".jpeg,.jpg,.png"
          type="file"
        />
        <button onClick={() => fileInputRef.current?.click()} type="button">
          Select image
        </button>
        <button type="submit">{id ? "Update room" : "Create room"}</button>
        <button onClick={() => navigate("/room/menu")} type="button">
          Back
        </button>
        {coverImageURL && (
          <div
            style={{ backgroundImage: `url(${coverImageURL})` }}
            className={classes.coverImage}
          />
        )}
        <ResMsg resMsg={resMsg} />
      </form>
    </ProtectedRoute>
  );
}
