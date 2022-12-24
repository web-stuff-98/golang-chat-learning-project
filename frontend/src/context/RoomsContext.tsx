import { useState, useContext, createContext } from "react";
import type { ReactNode } from "react";

export interface IRoom {
  ID: string;
  name: string;
  base64image?: string;
  author_id: string;
}

const RoomsContext = createContext<{
  rooms: IRoom[];
  updateRoomData: (data: Partial<IRoom>) => void;
  setAllRooms: (rooms: IRoom[]) => void;
  ownRooms: boolean;
  setOwnRooms: (to: boolean) => void;
}>({
  rooms: [],
  updateRoomData: () => {},
  setAllRooms: () => {},
  ownRooms: false,
  setOwnRooms: () => {},
});

export const RoomsProvider = ({ children }: { children: ReactNode }) => {
  const [rooms, setRooms] = useState<IRoom[]>([]);
  const [ownRooms, setOwnRooms] = useState(false);

  const updateRoomData = (data: Partial<IRoom>) => {
    setRooms((old) => {
      let newRooms = old;
      const i = old.findIndex((r) => r.ID === data.ID);
      if (i !== -1) {
        newRooms[i] = { ...newRooms[i], ...data };
      } else {
        newRooms = [...newRooms, data as IRoom];
      }
      return [...newRooms];
    });
  };

  const setAllRooms = (rooms: IRoom[]) => setRooms(rooms);

  return (
    <RoomsContext.Provider
      value={{ rooms, updateRoomData, setAllRooms, setOwnRooms, ownRooms }}
    >
      {children}
    </RoomsContext.Provider>
  );
};

export const useRooms = () => useContext(RoomsContext);
