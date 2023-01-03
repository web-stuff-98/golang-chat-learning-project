import { useUsers } from "../context/UsersContext";
import { IMsg } from "../routes/Room";
import classes from "../styles/pages/Room.module.scss";
import User from "./User";

import { useState, useRef, useEffect } from "react";
import type { CancelToken, CancelTokenSource } from "axios";
import axios from "axios";
import { getAttachmentImage } from "../services/rooms";
import { baseURL } from "../services/makeRequest";
import { ImSpinner8 } from "react-icons/im";
import { AiOutlineDownload } from "react-icons/ai";

export default function Message({
  msg,
  reverse,
}: {
  msg: IMsg;
  reverse: boolean;
}) {
  const { getUserData } = useUsers();

  return (
    <div
      style={reverse ? { alignItems: "flex-end" } : {}}
      className={classes.message}
    >
      <div
        className={classes.userAndTextContent}
        style={reverse ? { flexDirection: "row-reverse" } : {}}
      >
        <User
          uid={msg.uid}
          date={new Date(msg.timestamp)}
          reverse={reverse}
          user={getUserData(msg.uid)}
        />
        <div
          style={reverse ? { textAlign: "right" } : {}}
          className={classes.messageContent}
        >
          {msg.content}
        </div>
      </div>
      {msg.has_attachment && !msg.attachment_pending && (
        <>
          {msg.attachment_type === "image/jpeg" ? (
            <img
              src={`${baseURL}/api/attachment/image/${msg.ID!}`}
              className={classes.imageAttachment}
            />
          ) : (
            <a
              download={`${msg.ID}.${msg.attachment_type?.split("/")[1]}`}
              href={`${baseURL}/api/attachment/download/${msg.ID}`}
              aria-label="Download attachment"
              className={classes.downloadAttachment}
              style={reverse ? { flexDirection: "row-reverse" } : {}}
            >
              <AiOutlineDownload />
              Download attachment
            </a>
          )}
        </>
      )}
      {msg.attachment_pending && (
        <div className={classes.pending}>
          <ImSpinner8 className={classes.spinner} />
          Attachment pending...
        </div>
      )}
    </div>
  );
}
