import { makeRequest } from "./makeRequest";

const getRooms = (getOwnRooms: boolean) =>
  makeRequest(`/api/rooms${getOwnRooms ? "?own=true" : "?own=false"}`, {
    withCredentials: true,
  });

const getRoom = (id: string) =>
  makeRequest(`/api/room/${id}`, { withCredentials: true });

const createRoom = (data: { name: string }) =>
  makeRequest("/api/rooms", {
    method: "POST",
    withCredentials: true,
    data,
  });

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

const updateRoom = (id: string, data: { name: string }) =>
  makeRequest(`/api/rooms/${id}`, {
    method: "PATCH",
    data,
  });

export { getRoom, getRooms, createRoom, updateRoom, joinRoom, leaveRoom };
