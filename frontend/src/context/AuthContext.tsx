import { useState, useContext, createContext, useEffect } from "react";
import type { ReactNode } from "react";
import { makeRequest } from "../services/makeRequest";

export interface IUser {
  ID: string;
  username: string;
  base64pfp?: string;
}

const AuthContext = createContext<{
  user?: IUser;
  login: (username: string, password: string) => void;
  logout: () => void;
  deleteAccount: () => void;
  register: (username: string, password: string) => void;
  updateUserState: (user: Partial<IUser>) => void;
}>({
  user: undefined,
  login: () => {},
  register: () => {},
  logout: () => {},
  deleteAccount: () => {},
  updateUserState: () => {},
});

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<IUser>();

  const login = async (username: string, password: string) => {
    const res = await makeRequest("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json;charset=UTF-8" },
      data: { username, password },
      withCredentials: true,
    });
    setUser(res);
  };

  const register = async (username: string, password: string) => {
    const res = await makeRequest("/api/register", {
      method: "POST",
      headers: { "Content-Type": "application/json;charset=UTF-8" },
      data: { username, password },
      withCredentials: true,
    });
    setUser(res);
  };

  const logout = async () => {
    await makeRequest("/api/logout", {
      method: "POST",
      withCredentials: true,
    });
    setUser((_) => undefined);
  };

  const deleteAccount = async () => {
    await makeRequest("/api/deleteacc", {
      withCredentials: true,
      method: "POST",
    });
    setUser((_) => undefined);
  };

  useEffect(() => {
    makeRequest("/api/refresh", {
      withCredentials: true,
      method: "POST",
    })
      .then((data) => {
        setUser(data);
      })
      .catch((e) => {
        console.warn(e);
      });
  }, []);
  useEffect(() => {
    const i = setInterval(async () => {
      try {
        await makeRequest("/api/refresh", {
          withCredentials: true,
          method: "POST",
        });
      } catch (e) {
        console.error(e);
        setUser(undefined);
      }
      //Refresh token every 60 seconds. Token expires after 120 seconds.
    }, 60000);
    return () => {
      clearInterval(i);
    };
  }, [user]);

  const updateUserState = (user: Partial<IUser>) =>
    setUser((old) => {
      return { ...old, ...user } as IUser;
    });

  return (
    <AuthContext.Provider
      value={{ user, login, register, logout, deleteAccount, updateUserState }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
