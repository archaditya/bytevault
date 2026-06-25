import { Sidebar } from "./sidebar";
import { Navbar } from "./navbar";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-bg">
      <Sidebar />
      <div className="md:pl-60">
        <Navbar />
        <main className="px-4 py-6 md:px-6">{children}</main>
      </div>
    </div>
  );
}
