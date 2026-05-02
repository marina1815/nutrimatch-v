"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError, getCurrentSession, logoutUser } from "@/lib/api";
import { clearClientSession } from "@/lib/session";
import { CurrentSession } from "@/lib/types";

function getInitials(name: string, email: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return email.slice(0, 2).toUpperCase();
}

export function ProfileMenu() {
  const pathname = usePathname();
  const router = useRouter();
  const [session, setSession] = useState<CurrentSession | null>(null);
  const [open, setOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    let active = true;

    const loadSession = async () => {
      try {
        const current = await getCurrentSession();
        if (active) {
          setSession(current);
        }
      } catch (error) {
        if (active) {
          setSession(null);
        }
        if (!(error instanceof ApiError && error.status === 401)) {
          console.warn("Unable to load profile menu session", error);
        }
      }
    };

    void loadSession();

    return () => {
      active = false;
    };
  }, [pathname]);

  if (!session) {
    return null;
  }

  const initials = getInitials(session.fullName, session.email);

  const handleLogout = async () => {
    if (loggingOut) {
      return;
    }

    setLoggingOut(true);
    try {
      await logoutUser();
    } catch {
      clearClientSession();
    } finally {
      setOpen(false);
      setLoggingOut(false);
      router.push("/login");
      router.refresh();
    }
  };

  return (
    <div className="nm-profile-menu">
      <button
        type="button"
        className="nm-profile-trigger"
        aria-label="Ouvrir le menu profil"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span>{initials}</span>
      </button>

      {open && (
        <div className="nm-profile-popover" role="menu">
          <div className="nm-profile-summary">
            <div className="nm-profile-avatar">{initials}</div>
            <div>
              <strong>{session.fullName || "Profil NutriMatch"}</strong>
              <span>{session.email}</span>
            </div>
          </div>

          <div className="nm-profile-quick">
            <span>Profil: {session.hasProfile ? "complete" : "a completer"}</span>
            <span>Auth: {session.authMethod || "standard"}</span>
          </div>

          <div className="nm-profile-actions">
            <Link className="nm-link-btn" href="/profile" onClick={() => setOpen(false)}>
              Voir mes infos
            </Link>
            <Link className="nm-link-btn" href="/onboarding" onClick={() => setOpen(false)}>
              Modifier
            </Link>
            <button
              type="button"
              className="nm-link-btn nm-profile-logout"
              onClick={() => void handleLogout()}
              disabled={loggingOut}
            >
              {loggingOut ? "Deconnexion..." : "Se deconnecter"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
