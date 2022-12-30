import { makeRequest } from "./makeRequest";

const getRooms = (getOwnRooms: boolean) =>
  makeRequest(`/api/rooms${getOwnRooms ? "?own=true" : "?own=false"}`, {
    withCredentials: true,
  });

const getRoom = (id: string) =>
  makeRequest(`/api/room/${id}`, { withCredentials: true });

const getRoomImage = async (id: string) => {
  const data = await makeRequest(`/api/room/${id}/image`, {
    withCredentials: true,
    responseType: "arraybuffer",
  });
  const blob = new Blob([data], { type: "image/jpeg" });
  return URL.createObjectURL(blob);
};

const createRoom = (data: { name: string }) =>
  makeRequest("/api/room", {
    method: "POST",
    withCredentials: true,
    data,
  });

const updateRoom = (id: string, data: { name: string }) =>
  makeRequest(`/api/room/${id}`, {
    method: "PATCH",
    data,
    withCredentials: true,
  });

const uploadRoomImage = (id: string, file: File) => {
  const formData = new FormData();
  formData.append("file", file);
  return makeRequest(`/api/room/${id}/image`, {
    withCredentials: true,
    method: "POST",
    data: formData,
  });
};

const joinRoom = (id: string) =>
  makeRequest(`/api/room/${id}/join`, {
    method: "POST",
    withCredentials: true,
  });

const leaveRoom = (id: string) =>
  makeRequest(`/api/room/${id}/leave`, {
    method: "POST",
    withCredentials: true,
  });

export {
  getRoom,
  getRooms,
  createRoom,
  updateRoom,
  joinRoom,
  leaveRoom,
  uploadRoomImage,
  getRoomImage,
};
