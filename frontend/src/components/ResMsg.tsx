import formClasses from "../styles/FormClasses.module.scss";

export interface IResMsg {
  msg: string;
  err: boolean;
  pen: boolean;
}

export default function ResMsg({ resMsg }: { resMsg: IResMsg }) {
  return (
    <>
      {resMsg.msg && (
        <div
          className={resMsg.err ? formClasses.resMsgErr : formClasses.resMsg}
        >
          {resMsg.msg}
        </div>
      )}
    </>
  );
}
