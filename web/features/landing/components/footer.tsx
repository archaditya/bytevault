import Link from "next/link";
import { Box } from "lucide-react";

const columns = [
  {
    title: "Product",
    links: [
      { label: "Dashboard", href: "/dashboard" },
      { label: "Transfers", href: "/transfers" },
      { label: "Storage", href: "/storage" },
      { label: "Analytics", href: "/analytics" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "Documentation", href: "#" },
      { label: "API reference", href: "#" },
      { label: "Status", href: "#" },
      { label: "Changelog", href: "#" },
    ],
  },
  {
    title: "Company",
    links: [
      { label: "About", href: "#" },
      { label: "Blog", href: "#" },
      { label: "Careers", href: "#" },
      { label: "Contact", href: "#" },
    ],
  },
];

export function Footer() {
  return (
    <footer className="py-16">
      <div className="container">
        <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
          <div>
            <div className="flex items-center gap-2">
              <div className="flex h-6 w-6 items-center justify-center rounded-sm bg-accent">
                <Box className="h-3.5 w-3.5 text-white" />
              </div>
              <span className="text-[14px] font-semibold text-ink">ByteVault</span>
            </div>
            <p className="mt-3 max-w-[180px] text-[12px] text-ink-muted">
              Resumable file transfer for engineering teams.
            </p>
          </div>
          {columns.map((col) => (
            <div key={col.title}>
              <p className="label-eyebrow">{col.title}</p>
              <ul className="mt-3 flex flex-col gap-2">
                {col.links.map((link) => (
                  <li key={link.label}>
                    <Link href={link.href} className="text-[13px] text-ink-muted hover:text-ink">
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-12 flex flex-col items-center justify-between gap-3 border-t border-border pt-6 text-[12px] text-ink-faint sm:flex-row">
          <span>&copy; {new Date().getFullYear()} ByteVault, Inc.</span>
          <div className="flex gap-5">
            <Link href="#" className="hover:text-ink">Privacy</Link>
            <Link href="#" className="hover:text-ink">Terms</Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
