import { useRef, useState } from "react";
import type { ChangeEvent } from "react";
import User from "../components/User";
import classes from "../styles/pages/Settings.module.scss";
import { IUser, useAuth } from "../context/AuthContext";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { makeRequest } from "../services/makeRequest";
import ProtectedRoute from "./ProtectedRoute";
import { BsGearWide } from "react-icons/bs";
import { useModal } from "../context/ModalContext";

export default function Settings() {
  const { user, deleteAccount, updateUserState } = useAuth();
  const { openModal, closeModal } = useModal();

  const [file, setFile] = useState<File>();
  const fileRef = useRef<File>();

  const [resMsg, setResMsg] = useState<IResMsg>({
    msg: "",
    err: false,
    pen: false,
  });

  const handlePfpInput = (e: ChangeEvent<HTMLInputElement>) => {
    if (!e.target.files) return;
    const file = e.target.files![0];
    if (!file) return;
    setFile(file);
    fileRef.current = file;
    updatePfp(file.name);
  };

  const updatePfp = async (name: string) => {
    try {
      if (!fileRef.current) throw new Error("No image selected");
      openModal("Confirm", {
        err: false,
        pen: false,
        msg: `Are you sure you want to use ${name} as your profile picture?`,
        confirmationCallback: () => {
          setResMsg({ msg: "", err: false, pen: true });
          const formData = new FormData();
          formData.append("file", fileRef.current as File, "pfp");
          makeRequest("/api/updatepfp", {
            method: "POST",
            withCredentials: true,
            data: formData,
          })
            .then(async () => {
              try {
                const b64 = await new Promise<string>((resolve, reject) => {
                  const fr = new FileReader();
                  fr.readAsDataURL(fileRef.current!);
                  fr.onloadend = () => resolve(fr.result as string);
                  fr.onabort = () => reject("Aborted");
                  fr.onerror = () => reject("Error");
                });
                updateUserState({ base64pfp: b64 });
                setResMsg({ msg: "", err: false, pen: false });
              } catch (e) {
                setResMsg({ msg: `${e}`, err: true, pen: false });
              }
              closeModal();
            })
            .catch((e) => {
              setResMsg({ msg: `${e}`, err: true, pen: false });
              closeModal();
            });
        },
        cancellationCallback: () => {},
      });
    } catch (e) {
      setResMsg({ msg: `${e}`, err: true, pen: false });
    }
  };

  const hiddenPfpInputRef = useRef<HTMLInputElement>(null);
  return (
    <ProtectedRoute user={user}>
      <form className={classes.container}>
        <div className={classes.heading}>
          <BsGearWide />
          <h1>Settings</h1>
        </div>
        <hr />
        <input
          onChange={handlePfpInput}
          type="file"
          ref={hiddenPfpInputRef}
          accept=".jpeg,.jpg,.png"
        />
        <User
          uid={user?.ID!}
          user={
            {
              ...user,
              ...(file ? { base64pfp: URL.createObjectURL(file) } : {}),
            } as IUser
          }
          onClick={() => hiddenPfpInputRef.current?.click()}
        />
        <p>
          You can click on your profile picture to select a new image. It will
          update for other users automatically.
        </p>
        <button
          onClick={() => deleteAccount()}
          className={classes.deleteAccountButton}
          type="button"
        >
          Delete account
        </button>
        <ResMsg resMsg={resMsg} />
      </form>
    </ProtectedRoute>
  );
}
