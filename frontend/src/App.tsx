import Layout from "./components/layout/Layout";
import { AuthProvider } from "./context/AuthContext";
import { InterfaceProvider } from "./context/InterfaceContext";
import { ModalProvider } from "./context/ModalContext";
import { MouseProvider } from "./context/MouseContext";
import { RoomsProvider } from "./context/RoomsContext";
import { SocketProvider } from "./context/SocketContext";
import { UsersProvider } from "./context/UsersContext";

function App() {
  return (
    <InterfaceProvider>
      <MouseProvider>
        <ModalProvider>
          <AuthProvider>
            <RoomsProvider>
              <UsersProvider>
                <SocketProvider>
                  <Layout />
                </SocketProvider>
              </UsersProvider>
            </RoomsProvider>
          </AuthProvider>
        </ModalProvider>
      </MouseProvider>
    </InterfaceProvider>
  );
}

export default App;
