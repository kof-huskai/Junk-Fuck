import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
// JetBrains Mono — the single application UI font, bundled locally
// (offline-safe). Latin subsets only: the UI is English (Persian falls back
// to Vazirmatn below), so bundling every script subset would just bloat the
// EXE. Weights actually used: 400 (body), 500/600 (buttons, emphasis), 700
// (headings, bold values).
import "@fontsource/jetbrains-mono/latin-400.css";
import "@fontsource/jetbrains-mono/latin-500.css";
import "@fontsource/jetbrains-mono/latin-600.css";
import "@fontsource/jetbrains-mono/latin-700.css";
// Vazirmatn — the Persian UI font, bundled locally (offline-safe).
import "@fontsource/vazirmatn/400.css";
import "@fontsource/vazirmatn/500.css";
import "@fontsource/vazirmatn/600.css";
import "@fontsource/vazirmatn/700.css";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
