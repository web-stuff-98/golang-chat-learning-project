import { useSocket } from "../context/SocketContext";
import classes from "../styles/pages/Home.module.scss";

import { useState } from "react"
import type { FormEvent, ChangeEvent } from "react";

export default function Home() {
  const { socket } = useSocket();

  const [messageInput, setMessageInput] = useState("")

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log(messageInput)
    socket?.send(messageInput)
  };

  return (
    <div className={classes.container}>
      <div className={classes.messages}></div>
      <form onSubmit={handleSubmit}>
        <input onChange={(e:ChangeEvent<HTMLInputElement>) => setMessageInput(e.target.value)} type="text" />
        <button>Send</button>
      </form>
    </div>
  );
}
