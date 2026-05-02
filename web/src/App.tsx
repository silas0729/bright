import AdminPortal from "./AdminPortal";
import PublicSite from "./PublicSite";

export default function App() {
  if (window.location.pathname.startsWith("/admin")) {
    return <AdminPortal />;
  }
  return <PublicSite />;
}
