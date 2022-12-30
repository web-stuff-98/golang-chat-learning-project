import axios, { AxiosRequestConfig } from "axios";

const api = axios.create({
  baseURL: process.env.NODE_ENV === "development" ? "http://localhost:8080" : "https://golang-chat-learning-project.herokuapp.com",
});

export function makeRequest(url: string, options?: AxiosRequestConfig) {
  return api(url, options)
    .then((res) => res.data)
    .catch((error) =>
      Promise.reject(
        (error.response?.data.message ?? "Error").replace("Error: ", "")
      )
    );
}
