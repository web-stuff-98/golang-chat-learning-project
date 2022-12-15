import classes from "../styles/pages/RoomEditor.module.scss";
import formClasses from "../styles/FormClasses.module.scss";
import { useParams } from "react-router-dom";

import { useState } from "react";
import type { FormEvent, ChangeEvent } from "react";
import { createRoom, updateRoom } from "../services/rooms";
import ResMsg, { IResMsg } from "../components/ResMsg";

export default function RoomEditor() {
  const { id } = useParams();

  const [nameInput, setNameInput] = useState("");

  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      setResMsg({
        msg: id ? "Updating room" : "Creating room",
        err: false,
        pen: true,
      });
      const prom = id
        ? updateRoom(id, { name: nameInput })
        : createRoom({ name: nameInput });
      setResMsg({ msg: "", err: false, pen: false });
      const res = await prom;
    } catch (e) {
      setResMsg({ msg: `${e}`, pen: false, err: true });
    }
  };

  return (
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
      <button type="submit">{id ? "Update room" : "Create room"}</button>
      <ResMsg resMsg={resMsg} />
    </form>
  );
}
