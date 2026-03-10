import React from "react";
import { createRoot } from "react-dom/client";

function App() {
  return (
    <main style={{ fontFamily: "system-ui", padding: 16 }}>
      <h1>Opener NetDoor Desktop Client</h1>
      <p>TODO: user connect flow, protocol auto-select, diagnostics.</p>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);

