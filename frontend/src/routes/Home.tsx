import classes from "../styles/pages/Home.module.scss";

import ReactPlayer from "react-player";

export default function Home() {
  return (
    <div className={classes.container}>
      <h1>Go-Chat</h1>
      <h2>By Jason</h2>
      <hr />
      <h3>username : TestAcc1-TestAcc50</h3>
      <h3>password : Test1234! (with the !)</h3>
      <ReactPlayer
        controls
        width={"100%"}
        height={"auto"}
        url={"https://d2gt89ey9qb5n6.cloudfront.net/govid.mp4"}
      />
      <p>
        This is my chat app made using React and Go. It is my first Go project.
        It uses Fiber & MongoDB. It has refresh tokens, rooms and customizable
        profile pictures. Any new account and all its associated rooms and
        messages will be deleted automatically after 20 minutes. Updates to room
        and profile images will be shown right away for other users.
      </p>
    </div>
  );
}
