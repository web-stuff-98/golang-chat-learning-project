import { useLocation } from "react-router-dom";
import { useContext, createContext, useReducer, useEffect } from "react";
import type { ReactNode } from "react";

type LayoutContainerMode = "Modal" | "Feed" | "Full";

const initialState: State = {
  darkMode: false,
  dimensions: { width: 0, height: 0 },
  containerMode: "Full",
};

type State = {
  darkMode: boolean;
  dimensions: { width: number; height: number };
  containerMode: LayoutContainerMode;
};

const InterfaceContext = createContext<{
  state: State;
  dispatch: (action: Partial<State>) => void;
}>({
  state: initialState,
  dispatch: () => {},
});

const InterfaceReducer = (state: State, action: Partial<State>) => ({
  ...state,
  ...action,
});

export const InterfaceProvider = ({ children }: { children: ReactNode }) => {
  const location = useLocation();

  const [state, dispatch] = useReducer(InterfaceReducer, initialState);

  useEffect(() => {
    const handleResize = () =>
      dispatch({
        dimensions: { width: window.innerWidth, height: window.innerHeight },
      });
    handleResize();
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  useEffect(() => {
    if (!location) return;
    if (
      location.pathname === "/login" ||
      location.pathname === "/register" ||
      location.pathname === "/settings" ||
      location.pathname.includes("/room/") ||
      location.pathname === "/"
    ) {
      dispatch({ containerMode: "Modal" });
    } else {
      dispatch({ containerMode: "Feed" });
    }
  }, [location]);

  return (
    <InterfaceContext.Provider value={{ state, dispatch }}>
      {children}
    </InterfaceContext.Provider>
  );
};

export const useInterface = () => useContext(InterfaceContext);
