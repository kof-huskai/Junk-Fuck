import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
// Geist — the primary Latin UI font, bundled locally (offline-safe).
import "@fontsource/geist-sans/400.css";
import "@fontsource/geist-sans/500.css";
import "@fontsource/geist-sans/600.css";
import "@fontsource/geist-sans/700.css";
// Geist Mono — sparingly, for technical values (paths, versions, hashes).
// Latin subset only: the mono face is used for paths/versions, so bundling
// cyrillic/vietnamese/symbol subsets would just bloat the EXE.
import "@fontsource/geist-mono/latin-400.css";
import "@fontsource/geist-mono/latin-500.css";
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
