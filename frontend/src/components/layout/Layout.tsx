import classes from "./Layout.module.scss";
import Nav from "./Nav";

import type { CSSProperties } from "react";

import { Outlet } from "react-router-dom";
import { useInterface } from "../../context/InterfaceContext";

export default function Layout() {
  const { state: iState } = useInterface();

  const getStyle = () => {
    switch (iState.containerMode) {
      case "Feed": {
        const properties: CSSProperties = {
          width: "calc(100% - var(--horizontal-whitespace) * 2)",
          background: "var(--foreground)",
        };
        return properties;
      }
      case "Full": {
        const properties: CSSProperties = {
          width: "100%",
        };
        return properties;
      }
      case "Modal": {
        const properties: CSSProperties = {
          width: "17rem",
          height: "fit-content",
          maxWidth: "min(17rem, calc(100vw - 0.5rem))",
          background: "var(--foreground)",
          border: "1px solid var(--base-medium)",
          borderRadius: "var(--border-radius)",
          margin: "auto",
          boxShadow: "2px 2px 3px rgba(0,0,0,0.066)",
        };
        return properties;
      }
    }
  };

  return (
    <div className={classes.container}>
      <div className={classes.backgroundOuterContainer}>
        <div className={classes.backgroundInnerContainer}>
          <div aria-label="hidden" className={classes.background} />
          <div
            aria-label="hidden"
            style={{
              maskImage: `radial-gradient(circle at 50% 50%, transparent -50%, rgba(0,0,0,0.25) 86.66%)`,
              WebkitMaskImage: `radial-gradient(circle at 50% 50%, transparent -50%, rgba(0,0,0,0.25) 86.66%)`,
              ...(iState.darkMode
                ? { filter: "brightness(5.5) contrast(1.5) blur(3px)" }
                : {}),
            }}
            className={classes.backgroundHover}
          />
        </div>
      </div>
      <Nav />
      <main style={getStyle()}>
        <Outlet />
      </main>
    </div>
  );
}
