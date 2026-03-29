import { BrowserRouter, Routes, Route, NavLink, useLocation } from "react-router-dom";
import { AuthProvider } from "@/context/AuthContext";
import { ToastProvider } from "@/components/ui/toast";
import { Auth401Dialog } from "@/components/ui/Auth401Dialog";
import { DashboardPage } from "@/pages/DashboardPage";
import { SubscriptionsPage } from "@/pages/SubscriptionsPage";
import { SubscriptionDetailPage } from "@/pages/SubscriptionDetailPage";
import { CollectionsPage } from "@/pages/CollectionsPage";
import { CollectionDetailPage } from "@/pages/CollectionDetailPage";
import { NodesPage } from "@/pages/NodesPage";
import { NodeDetailPage } from "@/pages/NodeDetailPage";
import { EndpointsPage } from "@/pages/EndpointsPage";
import { RulesPage } from "@/pages/RulesPage";
import { HealthPage } from "@/pages/HealthPage";
import { SettingsPage } from "@/pages/SettingsPage";
import { ProfilesPage } from "@/pages/ProfilesPage";
import { ProfileEditorPage } from "@/pages/ProfileEditorPage";
import {
  LayoutDashboard,
  BookMarked,
  Server,
  Plug,
  ListFilter,
  HeartPulse,
  Settings,
  LayoutList,
  FolderOpen,
  ChevronDown,
  Menu,
  X,
} from "lucide-react";
import { useState, useEffect } from "react";

function ComingSoon({ name }: { name: string }) {
  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">{name}</h1>
      <p className="text-sm" style={{ color: "var(--color-muted-foreground)" }}>
        Coming soon…
      </p>
    </div>
  );
}

// isIconOnly: when true, render icon only (tablet sidebar collapsed to 64px)
function navLinkClass(isActive: boolean, isIconOnly: boolean) {
  return [
    "flex items-center rounded-lg transition-colors",
    isIconOnly
      ? "justify-center py-3 w-full min-h-[42px]"
      : "gap-2.5 px-3 py-2.5 text-[13px] font-medium min-h-[42px]",
    isActive
      ? "bg-[var(--color-primary-bg)] text-[var(--color-primary-light)] border-l-2 border-[var(--color-primary)]"
      : "text-[var(--color-text-muted)] hover:bg-[var(--color-bg-accent)] hover:text-[var(--color-text-primary)]",
  ].join(" ");
}

function NavItem({
  to,
  label,
  icon: Icon,
  end,
  iconOnly = false,
  onClick,
}: {
  to: string;
  label: string;
  icon: React.ElementType;
  end?: boolean;
  iconOnly?: boolean;
  onClick?: () => void;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      title={iconOnly ? label : undefined}
      className={({ isActive }) => navLinkClass(isActive, iconOnly)}
      onClick={onClick}
    >
      <Icon className="h-[15px] w-[15px] shrink-0" />
      {!iconOnly && label}
    </NavLink>
  );
}

function CollapsibleSection({
  label,
  children,
  defaultOpen = true,
  hidden = false,
}: {
  label: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
  hidden?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  if (hidden) return <div className="flex flex-col gap-0.5">{children}</div>;
  return (
    <div>
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between px-3 pt-4 pb-1 text-[10px] font-bold uppercase tracking-[0.08em] transition-colors text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
      >
        {label}
        <ChevronDown
          className="h-3 w-3 transition-transform"
          style={{ transform: open ? "rotate(0deg)" : "rotate(-90deg)" }}
        />
      </button>
      {open && <div className="flex flex-col gap-0.5">{children}</div>}
    </div>
  );
}

function LogoBadge({ iconOnly = false }: { iconOnly?: boolean }) {
  return (
    <div className={iconOnly ? "flex items-center justify-center w-full mb-4" : "flex items-center gap-2.5 mb-6 px-1"}>
      <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-[var(--color-primary)] shrink-0">
        <span className="text-[13px] font-bold text-[#1C1917]">S</span>
      </div>
      {!iconOnly && (
        <span className="text-[15px] font-bold text-[var(--color-text-primary)] tracking-tight">SubHub</span>
      )}
    </div>
  );
}

// SidebarNav: the nav items, shared between overlay and persistent sidebar
function SidebarNav({ iconOnly = false, onNavClick }: { iconOnly?: boolean; onNavClick?: () => void }) {
  return (
    <nav className="flex flex-col gap-0.5">
      <NavItem to="/" label="Dashboard" icon={LayoutDashboard} end iconOnly={iconOnly} onClick={onNavClick} />

      <CollapsibleSection label="Sources" hidden={iconOnly}>
        <NavItem to="/subscriptions" label="Subscriptions" icon={BookMarked} iconOnly={iconOnly} onClick={onNavClick} />
        <NavItem to="/collections" label="Collections" icon={FolderOpen} iconOnly={iconOnly} onClick={onNavClick} />
      </CollapsibleSection>

      <CollapsibleSection label="Output" hidden={iconOnly}>
        <NavItem to="/endpoints" label="Endpoints" icon={Plug} iconOnly={iconOnly} onClick={onNavClick} />
        <NavItem to="/profiles" label="Profiles" icon={LayoutList} iconOnly={iconOnly} onClick={onNavClick} />
      </CollapsibleSection>

      <CollapsibleSection label="Advanced" defaultOpen={false} hidden={iconOnly}>
        <NavItem to="/nodes" label="Nodes" icon={Server} iconOnly={iconOnly} onClick={onNavClick} />
        <NavItem to="/rules" label="Rules" icon={ListFilter} iconOnly={iconOnly} onClick={onNavClick} />
      </CollapsibleSection>

      <CollapsibleSection label="System" hidden={iconOnly}>
        <NavItem to="/health" label="Health" icon={HeartPulse} iconOnly={iconOnly} onClick={onNavClick} />
        <NavItem to="/settings" label="Settings" icon={Settings} iconOnly={iconOnly} onClick={onNavClick} />
      </CollapsibleSection>
    </nav>
  );
}

const PAGE_TITLES: Record<string, string> = {
  "/": "Dashboard",
  "/subscriptions": "Subscriptions",
  "/collections": "Collections",
  "/endpoints": "Endpoints",
  "/profiles": "Profiles",
  "/nodes": "Nodes",
  "/rules": "Rules",
  "/health": "Health",
  "/settings": "Settings",
};

function getPageTitle(pathname: string): string {
  if (PAGE_TITLES[pathname]) return PAGE_TITLES[pathname];
  const prefix = "/" + pathname.split("/")[1];
  return PAGE_TITLES[prefix] ?? "SubHub";
}

function MobileHeader({
  onMenuClick,
  pageTitle,
}: {
  onMenuClick: () => void;
  pageTitle: string;
}) {
  return (
    <header className="md:hidden fixed top-0 left-0 right-0 z-40 flex items-center h-14 px-4 gap-3 bg-[var(--color-bg-sidebar)] border-b border-[var(--color-border)]">
      <button
        onClick={onMenuClick}
        aria-label="Open menu"
        className="flex items-center justify-center w-9 h-9 rounded-lg text-[var(--color-text-primary)] hover:bg-[var(--color-bg-accent)] transition-colors shrink-0"
      >
        <Menu className="h-5 w-5" />
      </button>
      <div className="flex items-center gap-2 shrink-0">
        <div className="flex items-center justify-center w-7 h-7 rounded-md bg-[var(--color-primary)]">
          <span className="text-[12px] font-bold text-[#1C1917]">S</span>
        </div>
        <span className="text-[14px] font-bold text-[var(--color-text-primary)] tracking-tight">SubHub</span>
      </div>
      <div className="flex-1" />
      <span className="text-[13px] font-medium text-[var(--color-text-primary)] truncate max-w-[120px]">
        {pageTitle}
      </span>
    </header>
  );
}

function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const location = useLocation();

  // Close mobile overlay on route change
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSidebarOpen(false);
  }, [location.pathname]);

  return (
    <div className="flex min-h-screen">
      <MobileHeader
        onMenuClick={() => setSidebarOpen((v) => !v)}
        pageTitle={getPageTitle(location.pathname)}
      />

      {/* ── Mobile: overlay backdrop ── */}
      {sidebarOpen && (
        <div
          className="md:hidden fixed inset-0 z-40 bg-black/60"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* ── Mobile: slide-in drawer (overlay, <768px only) ── */}
      <aside
        className={[
          "md:hidden fixed top-0 left-0 z-50 h-full w-[280px] flex flex-col px-3 py-5 bg-[var(--color-bg-sidebar)] border-r border-[var(--color-border)] transition-transform duration-200 ease-in-out",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
        ].join(" ")}
      >
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2.5">
            <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-[var(--color-primary)]">
              <span className="text-[13px] font-bold text-[#1C1917]">S</span>
            </div>
            <span className="text-[15px] font-bold text-[var(--color-text-primary)] tracking-tight">SubHub</span>
          </div>
          <button
            onClick={() => setSidebarOpen(false)}
            aria-label="Close menu"
            className="flex items-center justify-center w-8 h-8 rounded-lg text-[var(--color-text-muted)] hover:bg-[var(--color-bg-accent)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <SidebarNav onNavClick={() => setSidebarOpen(false)} />
      </aside>

      {/* ── Tablet: icon-only sidebar (768px–1024px) ── */}
      <aside className="hidden md:flex lg:hidden w-16 shrink-0 flex-col items-center py-5 gap-0.5 border-r border-[var(--color-border)] bg-[var(--color-bg-sidebar)]">
        <LogoBadge iconOnly />
        <SidebarNav iconOnly />
      </aside>

      {/* ── Desktop: full sidebar (>=1024px) ── */}
      <aside className="hidden lg:flex w-56 shrink-0 flex-col px-3 py-5 gap-0.5 border-r border-[var(--color-border)] bg-[var(--color-bg-sidebar)]">
        <LogoBadge />
        <SidebarNav />
      </aside>

      {/* ── Main content ── */}
      <main className="flex-1 pt-14 md:pt-0 p-4 md:p-6 lg:p-10 overflow-auto">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/subscriptions" element={<SubscriptionsPage />} />
          <Route path="/subscriptions/:id" element={<SubscriptionDetailPage />} />
          <Route path="/collections" element={<CollectionsPage />} />
          <Route path="/collections/:id" element={<CollectionDetailPage />} />
          <Route path="/nodes" element={<NodesPage />} />
          <Route path="/nodes/:id" element={<NodeDetailPage />} />
          <Route path="/endpoints" element={<EndpointsPage />} />
          <Route path="/rules" element={<RulesPage />} />
          <Route path="/profiles" element={<ProfilesPage />} />
          <Route path="/profiles/:id" element={<ProfileEditorPage />} />
          <Route path="/health" element={<HealthPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<ComingSoon name="Not Found" />} />
        </Routes>
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <Auth401Dialog />
        <BrowserRouter>
          <Layout />
        </BrowserRouter>
      </ToastProvider>
    </AuthProvider>
  );
}
