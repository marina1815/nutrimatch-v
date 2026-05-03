"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { logoutUser } from "@/lib/api";
import { clearClientSession } from "@/lib/session";

export default function SignOutPage() {
  const router = useRouter();

  useEffect(() => {
    let cancelled = false;

    const signOut = async () => {
      try {
        await logoutUser();
      } catch {
        clearClientSession();
      } finally {
        if (!cancelled) {
          router.replace("/login");
        }
      }
    };

    void signOut();

    return () => {
      cancelled = true;
    };
  }, [router]);

  return (
    <main className="nm-page">
      <div className="nm-card">
        <h1 className="nm-title">Deconnexion</h1>
        <p className="nm-sub">Revocation de la session en cours...</p>
      </div>
    </main>
  );
}
