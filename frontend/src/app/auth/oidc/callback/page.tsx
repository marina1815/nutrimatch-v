"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getCurrentSession } from "@/lib/api";
import { clearClientSession, setAccessToken, setCurrentProfileId } from "@/lib/session";

function normalizeNextPath(input: string | null) {
  if (!input || !input.startsWith("/") || input.startsWith("//")) {
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
      const fragment = new URLSearchParams(window.location.hash.replace(/^#/, ""));
      const accessToken = fragment.get("access_token");
      const nextPath = normalizeNextPath(searchParams.get("next"));

      if (!accessToken) {
        clearClientSession();
        if (active) setError("Le retour OpenID Connect est incomplet.");
        return;
      }

      setAccessToken(accessToken);
      window.history.replaceState({}, document.title, window.location.pathname + window.location.search);

      let target = nextPath;
      try {
        const session = await getCurrentSession();
        if (session.profileId) {
          setCurrentProfileId(session.profileId);
        }
        if (!session.hasProfile && target === "/results") {
          target = "/onboarding";
        }
      } catch {
        if (target === "/results") {
          target = "/onboarding";
        }
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
      <section className="nm-results-shell">
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
          <section className="nm-results-shell">
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
