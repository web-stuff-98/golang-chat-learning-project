import classes from "../styles/pages/Home.module.scss";

export default function Home() {
  return (
    <div className={classes.container}>
        <h1>Go-Chat</h1>
        <p>This is my chat app made using React and Go. It is my first Go project. It uses Fiber & MongoDB. It has refresh tokens, rooms and customizable profile pictures. Any new account and all its associated rooms and messages will be deleted automatically after 20 minutes. Updates to room and profile images will be shown right away for other users.</p>
    </div>
  );
}
