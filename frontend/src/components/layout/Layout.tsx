import classes from "./Layout.module.scss";
import Nav from "./Nav";

import type { CSSProperties } from "react"

import { Outlet } from "react-router-dom";
import { useInterface } from "../../context/InterfaceContext";

export default function Layout() {
  const { state: iState } = useInterface();

  const getStyle = () => {
    switch (iState.containerMode) {
        case "Feed": {
            const properties:CSSProperties = {
                width: "calc(100% - var(--horizontal-whitespace) * 2)",
                background:"var(--foreground)"
            }
            return properties
        }
        case "Full": {
            const properties:CSSProperties = {
                width: "100%"
            }
            return properties

        }
        case "Modal": {
            const properties:CSSProperties = {
                width: "fit-content",
                height:"fit-content",
                background:"var(--foreground)",
                border: "1px solid var(--base-medium)",
                borderRadius: "var(--border-radius)",
                margin:"auto"
            }
            return properties
        }
    }
  };

  return (
    <div className={classes.container}>
      <Nav />
      <main style={getStyle()}>
        <Outlet />
      </main>
    </div>
  );
}
