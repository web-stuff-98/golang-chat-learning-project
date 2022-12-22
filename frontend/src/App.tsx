import Layout from "./components/layout/Layout";
import { AuthProvider } from "./context/AuthContext";
import { InterfaceProvider } from "./context/InterfaceContext";
import { SocketProvider } from "./context/SocketContext";
import { UsersProvider } from "./context/UsersContext";

function App() {
  return (
    <InterfaceProvider>
      <AuthProvider>
        <SocketProvider>
          <UsersProvider>
            <Layout />
          </UsersProvider>
        </SocketProvider>
      </AuthProvider>
    </InterfaceProvider>
  );
}

export default App;
