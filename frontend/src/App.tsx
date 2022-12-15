import Layout from "./components/layout/Layout";
import { AuthProvider } from "./context/AuthContext";
import { InterfaceProvider } from "./context/InterfaceContext";
import { SocketProvider } from "./context/SocketContext";

function App() {
  return (
    <InterfaceProvider>
      <AuthProvider>
        <SocketProvider>
          <Layout />
        </SocketProvider>
      </AuthProvider>
    </InterfaceProvider>
  );
}

export default App;
