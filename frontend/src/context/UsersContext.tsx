import {
  useState,
  useContext,
  createContext,
  useEffect,
  useCallback,
} from "react";
import type { ReactNode } from "react";
import { IUser, useAuth } from "./AuthContext";
import { makeRequest } from "../services/makeRequest";

type DisappearedUser = {
  uid: string;
  disappearedAt: Date;
};

const UsersContext = createContext<{
  users: IUser[];
  getUserData: (uid: string) => IUser;
  cacheUserData: (uid: string, force?: boolean) => void;

  userEnteredView: (uid: string) => void;
  userLeftView: (uid: string) => void;

  deleteUser: (id: string) => void;

  updateUserData: (data: Omit<Partial<IUser>, "event_type">) => void;
}>({
  users: [],
  getUserData: () => ({ username: "", ID: "" }),
  cacheUserData: () => {},

  userEnteredView: () => {},
  userLeftView: () => {},

  deleteUser: () => {},

  updateUserData: () => {},
});

export const UsersProvider = ({ children }: { children: ReactNode }) => {
  const { user: currentUser } = useAuth();

  const [users, setUsers] = useState<IUser[]>([]);

  const getUserData = useCallback(
    (uid: string) => {
      if (currentUser && uid === currentUser.ID) return currentUser;
      return users.find((u) => u.ID === uid) as IUser;
    },
    [users]
  );

  const cacheUserData = async (uid: string, force?: boolean) => {
    const foundIndex = users.findIndex((u) => u.ID === uid);
    if (foundIndex !== -1 && !force) return;
    try {
      if (currentUser && uid === currentUser.ID) return;
      const data = await makeRequest(`/api/user/${uid}`, {
        withCredentials: true,
      });
      setUsers((old) => [...old.filter((u) => u.ID !== uid), data]);
    } catch (e) {
      console.error("Could not get data for user " + uid);
    }
  };

  const updateUserData = (data: Partial<IUser>) => {
    setUsers((old) => {
      let newdata = old;
      const i = newdata.findIndex((u) => u.ID === data.ID!);
      if (i === -1) return old;
      newdata[i] = { ...newdata[i], ...data };
      return [...newdata];
    });
  };

  const deleteUser = (id: string) => {
    setVisibleUsers((o) => [...o.filter((u) => u !== id)]);
    setDisappearedUsers((o) => [...o.filter((u) => u.uid !== id)]);
    setUsers((o) => [...o.filter((o) => o.ID !== id)]);
  };

  const [visibleUsers, setVisibleUsers] = useState<string[]>([]);
  const [disappearedUsers, setDisappearedUsers] = useState<DisappearedUser[]>(
    []
  );
  const userEnteredView = (uid: string) => {
    if (currentUser && currentUser.ID === uid) return;
    setVisibleUsers((p) => [...p, uid]);
    setDisappearedUsers((p) => [...p.filter((u) => u.uid !== uid)]);
  };
  const userLeftView = (uid: string) => {
    if (currentUser && currentUser.ID === uid) return;
    const visibleCount =
      visibleUsers.filter((visibleUid) => visibleUid === uid).length - 1;
    if (visibleCount === 0) {
      setVisibleUsers((p) => [...p.filter((visibleUid) => visibleUid !== uid)]);
      setDisappearedUsers((p) => [
        ...p.filter((p) => p.uid !== uid),
        {
          uid,
          disappearedAt: new Date(),
        },
      ]);
    } else {
      setVisibleUsers((p) => {
        //instead of removing all matching UIDs, remove only one, because we need to retain the duplicates
        let newVisibleUsers = p;
        newVisibleUsers.splice(
          p.findIndex((vuid) => vuid === uid),
          1
        );
        return [...newVisibleUsers];
      });
    }
  };
  useEffect(() => {
    const i = setInterval(() => {
      const usersDisappeared30SecondsAgo = disappearedUsers
        .filter(
          (du) => new Date().getTime() - du.disappearedAt.getTime() > 30000
        )
        .map((du) => du.uid);
      setUsers((p) => [
        ...p.filter((u) => !usersDisappeared30SecondsAgo.includes(u.ID)),
      ]);
      setDisappearedUsers((p) => [
        ...p.filter((u) => !usersDisappeared30SecondsAgo.includes(u.uid)),
      ]);
    }, 5000);
    return () => {
      clearInterval(i);
    };
  }, [disappearedUsers]);

  return (
    <UsersContext.Provider
      value={{
        users,
        getUserData,
        cacheUserData,
        userEnteredView,
        userLeftView,
        updateUserData,
        deleteUser,
      }}
    >
      {children}
    </UsersContext.Provider>
  );
};

export const useUsers = () => useContext(UsersContext);
