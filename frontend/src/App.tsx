import Layout from "./components/layout/Layout";
import { AuthProvider } from "./context/AuthContext";
import { InterfaceProvider } from "./context/InterfaceContext";
import { RoomsProvider } from "./context/RoomsContext";
import { SocketProvider } from "./context/SocketContext";
import { UsersProvider } from "./context/UsersContext";

function App() {
  return (
    <InterfaceProvider>
      <AuthProvider>
        <RoomsProvider>
          <UsersProvider>
            <SocketProvider>
              <Layout />
            </SocketProvider>
          </UsersProvider>
        </RoomsProvider>
      </AuthProvider>
    </InterfaceProvider>
  );
}

export default App;
