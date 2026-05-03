"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getCurrentSession } from "@/lib/api";
import { clearClientSession } from "@/lib/session";

const SAFE_NEXT_PATHS = new Set(["/results", "/profile", "/onboarding"]);

function normalizeNextPath(input: string | null) {
  if (!input || !input.startsWith("/") || input.startsWith("//") || !SAFE_NEXT_PATHS.has(input)) {
    return "/results";
  }
  return input;
}

function OIDCCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    const completeLogin = async () => {
      const nextPath = normalizeNextPath(searchParams.get("next"));

      window.history.replaceState({}, document.title, window.location.pathname + window.location.search);

      let target = nextPath;
      try {
        const session = await getCurrentSession();
        if (!session.hasProfile && target === "/results") {
          target = "/onboarding";
        }
      } catch {
        clearClientSession();
        if (active) setError("La session OpenID Connect n'a pas pu etre finalisee.");
        return;
      }

      if (active) {
        router.replace(target);
      }
    };

    void completeLogin();
    return () => {
      active = false;
    };
  }, [router, searchParams]);

  return (
    <main className="nm-page">
      <section className="nm-card">
        <h1 className="nm-title">Connexion securisee en cours</h1>
        <p className="nm-sub">
          {error || "Nous finalisons votre session et rechargeons votre espace NutriMatch."}
        </p>
      </section>
    </main>
  );
}

export default function OIDCCallbackPage() {
  return (
    <Suspense
      fallback={
        <main className="nm-page">
          <section className="nm-card">
            <h1 className="nm-title">Connexion securisee en cours</h1>
            <p className="nm-sub">
              Nous finalisons votre session et rechargeons votre espace NutriMatch.
            </p>
          </section>
        </main>
      }
    >
      <OIDCCallbackContent />
    </Suspense>
  );
}
