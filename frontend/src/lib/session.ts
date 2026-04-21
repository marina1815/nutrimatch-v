import { UserProfile } from "@/lib/types";

const ACCESS_TOKEN_KEY = "nutrimatch-access-token";
const PROFILE_DRAFT_KEY = "nutrimatch-profile-draft";
const PROFILE_ID_KEY = "nutrimatch-profile-id";

function getSessionStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null;
  }

  return window.sessionStorage;
}

export function getAccessToken(): string | null {
  return getSessionStorage()?.getItem(ACCESS_TOKEN_KEY) ?? null;
}

export function setAccessToken(token: string): void {
  getSessionStorage()?.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearAccessToken(): void {
  getSessionStorage()?.removeItem(ACCESS_TOKEN_KEY);
}

export function getCurrentProfileId(): string | null {
  return getSessionStorage()?.getItem(PROFILE_ID_KEY) ?? null;
}

export function setCurrentProfileId(profileId: string): void {
  getSessionStorage()?.setItem(PROFILE_ID_KEY, profileId);
}

export function clearCurrentProfileId(): void {
  getSessionStorage()?.removeItem(PROFILE_ID_KEY);
}

export function getDraftProfile(): UserProfile | null {
  const raw = getSessionStorage()?.getItem(PROFILE_DRAFT_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as UserProfile;
  } catch {
    clearDraftProfile();
    return null;
  }
}

export function setDraftProfile(profile: UserProfile): void {
  getSessionStorage()?.setItem(PROFILE_DRAFT_KEY, JSON.stringify(profile));
}

export function clearDraftProfile(): void {
  getSessionStorage()?.removeItem(PROFILE_DRAFT_KEY);
}

export function clearClientSession(): void {
  clearAccessToken();
  clearCurrentProfileId();
  clearDraftProfile();
}
