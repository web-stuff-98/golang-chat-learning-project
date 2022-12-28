import { useLocation } from "react-router-dom";
import { useContext, createContext, useReducer, useEffect } from "react";
import type { ReactNode } from "react";

type LayoutContainerMode = "Modal" | "Feed" | "Full";

const initialState: State = {
  darkMode: false,
  dimensions: { width: 0, height: 0 },
  containerMode: "Full",
};

function lerp(value1: number, value2: number, amount: number) {
  amount = amount < 0 ? 0 : amount;
  amount = amount > 1 ? 1 : amount;
  return value1 + (value2 - value1) * amount;
}

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
    const handleResize = () => {
      const lo = 700;
      const hi = 1024;
      const a = (Math.min(hi, Math.max(window.innerWidth, lo)) - lo) / (hi - lo);
      const v = lerp(window.innerWidth / 6 / 2, window.innerWidth / 2 / 2, Math.pow(a, 0.8));
      console.log(v)
      document.documentElement.style.setProperty(
        "--horizontal-whitespace",
        `${window.innerWidth < lo ? 0 : v}px`
      );
      dispatch({
        dimensions: { width: window.innerWidth, height: window.innerHeight },
      });
    };
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