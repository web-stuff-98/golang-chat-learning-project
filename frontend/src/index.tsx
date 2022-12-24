import ReactDOM from "react-dom/client";
import "./index.css";
import App from "./App";

import { Routes, Route, BrowserRouter } from "react-router-dom";
import Home from "./routes/Home";
import Login from "./routes/Login";
import Register from "./routes/Register";
import RoomEditor from "./routes/RoomEditor";
import RoomMenu from "./routes/RoomMenu";
import RoomList from "./routes/RoomList";
import Room from "./routes/Room";
import Settings from "./routes/Settings";

const root = ReactDOM.createRoot(
  document.getElementById("root") as HTMLElement
);
root.render(
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />}>
          <Route index element={<Home />} />
          <Route path="login" element={<Login />} />
          <Route path="register" element={<Register />} />
          <Route path="settings" element={<Settings />} />
          <Route path="room/edit" element={<RoomEditor />} />
          <Route path="room/edit/:id" element={<RoomEditor />} />
          <Route path="room/menu" element={<RoomMenu />} />
          <Route path="room/list" element={<RoomList />} />
          <Route path="room/:id" element={<Room />} />
        </Route>
      </Routes>
    </BrowserRouter>
);
