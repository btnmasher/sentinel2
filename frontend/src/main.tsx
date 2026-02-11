import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "@/app/App";
import "react-day-picker/dist/style.css";
import "@/styles.css";

const applyStoredTheme = () => {
  try {
    const raw = localStorage.getItem("intel-map-config/settings");
    if (!raw) {
      document.documentElement.setAttribute("data-theme", "sentinel");
      return;
    }
    const parsed = JSON.parse(raw);
    const theme = parsed?.state?.settings?.theme;
    if (typeof theme === "string" && theme.length > 0) {
      document.documentElement.setAttribute("data-theme", theme);
      return;
    }
  } catch {
    // Ignore invalid localStorage
  }
  document.documentElement.setAttribute("data-theme", "sentinel");
};

applyStoredTheme();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
);
