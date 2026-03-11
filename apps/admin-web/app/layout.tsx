import "./globals.css";

export const metadata = {
  title: "Opener NetDoor Admin",
  description: "Opener NetDoor control-plane admin console",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="dark">
      <body>{children}</body>
    </html>
  );
}
