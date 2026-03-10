import React from "react";
import { createRoot } from "react-dom/client";

function App() {
  return (
    <main style={{ fontFamily: "system-ui", padding: 16 }}>
      <h1>Opener NetDoor Manager Desktop</h1>
      <p>TODO: Implement server list, key lifecycle, audit and analytics views.</p>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);

