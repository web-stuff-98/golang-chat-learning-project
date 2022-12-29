import { useInterface } from "../../context/InterfaceContext";

import classes from "./Layout.module.scss";

export default function Header() {
  const { state: iState, dispatch: iDispatch } = useInterface();

  return (
    <header>
      <div className={classes.logoImg}>
        <img src="go-mascot-wiki-darkmode.png" />
        Go-Chat
      </div>
      <button
        aria-label="Toggle dark mode"
        onClick={() => iDispatch({ darkMode: !iState.darkMode })}
        className={classes.darkToggle}
      >
        {iState.darkMode ? "Dark mode" : "Light mode"}
      </button>
    </header>
  );
}
