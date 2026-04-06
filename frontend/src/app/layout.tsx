import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "NutriMatch — Recommandations de repas personnalisées",
  description:
    "Trouvez des repas adaptés à votre profil nutritionnel, votre mode de vie et vos contraintes alimentaires.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="fr" data-scroll-behavior="smooth">
      <body>{children}</body>
    </html>
  );
}