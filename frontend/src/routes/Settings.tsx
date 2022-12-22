import { useRef, useState } from "react";
import type { ChangeEvent } from "react";
import User from "../components/User";
import classes from "../styles/pages/Settings.module.scss";
import { IUser, useAuth } from "../context/AuthContext";
import ResMsg, { IResMsg } from "../components/ResMsg";
import { makeRequest } from "../services/makeRequest";

export default function Settings() {
  const { user, updateUserState } = useAuth();

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
    updatePfp();
  };

  const updatePfp = async () => {
    try {
      if (!fileRef.current) throw new Error("No image selected");
      setResMsg({ msg: "", err: false, pen: true });
      const formData = new FormData();
      formData.append("file", fileRef.current, "pfp");
      await makeRequest("/api/updatepfp", {
        method: "POST",
        withCredentials: true,
        data: formData,
      });
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
  };

  const hiddenPfpInputRef = useRef<HTMLInputElement>(null);
  return (
    <form className={classes.container}>
      <h1>Settings</h1>
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
      <p>You can click on your profile to select a new image.</p>
      <ResMsg resMsg={resMsg} />
    </form>
  );
}
