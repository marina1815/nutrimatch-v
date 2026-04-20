const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

// Read the CSRF token directly from the cookie (not HttpOnly, so JS can access it).
// This ensures we always use the latest token even after login/register refreshes it.
function readCSRFFromCookie(): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(/(?:^|;\s*)nm_csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : "";
}

// Ensure we have a CSRF token. First checks the cookie, then fetches a new one.
async function ensureCSRF(): Promise<{ name: string; token: string }> {
  if (typeof window === "undefined") return { name: "X-CSRF-Token", token: "" };

  // Always prefer the cookie value — it's the source of truth
  const cookieToken = readCSRFFromCookie();
  if (cookieToken) {
    return { name: "X-CSRF-Token", token: cookieToken };
  }

  // No cookie yet — fetch a fresh one
  const res = await fetch(`${API_URL}/api/v1/auth/csrf`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Failed to initialize security context (CSRF)");

  const data = await res.json();
  return { name: data.header_name || "X-CSRF-Token", token: data.csrf_token };
}

async function getHeaders(includeAuth = true) {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  // Add CSRF
  try {
    const csrf = await ensureCSRF();
    if (csrf.token) {
      headers[csrf.name] = csrf.token;
    }
  } catch (err) {
    console.warn("CSRF fetch failed:", err);
  }

  // Add Auth
  if (includeAuth && typeof window !== "undefined") {
    const token = localStorage.getItem("nutrimatch-token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  return headers;
}

export async function loginUser(payload: { email: string; password: string }) {
  const headers = await getHeaders(false);
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.error || error.message || "Login failed");
  }
  const data = await res.json();

  // Store the access token for subsequent authenticated requests
  if (data.access_token && typeof window !== "undefined") {
    localStorage.setItem("nutrimatch-token", data.access_token);
  }

  return data;
}

export async function registerUser(payload: {
  name: string;
  email: string;
  password: string;
}) {
  const headers = await getHeaders(false);
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.error || error.message || "Registration failed");
  }
  const data = await res.json();

  // Store the access token for subsequent authenticated requests
  if (data.access_token && typeof window !== "undefined") {
    localStorage.setItem("nutrimatch-token", data.access_token);
  }

  return data;
}

export async function saveProfile(profile: unknown) {
  const headers = await getHeaders(true);
  const res = await fetch(`${API_URL}/api/v1/profile`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify(profile),
  });

  if (!res.ok) {
    if (res.status === 401) {
      if (typeof window !== "undefined") {
        localStorage.removeItem("nutrimatch-token");
        window.location.href = "/login";
      }
    }
    const error = await res.json().catch(() => ({}));
    throw new Error(error.error || error.message || "Failed to submit profile");
  }
  return res.json();
}

export async function getRecommendations(profileId: string) {
  const headers = await getHeaders(true);
  const res = await fetch(`${API_URL}/api/v1/recommendations/${profileId}`, {
    headers,
    credentials: "include",
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({}));
    throw new Error(error.error || error.message || `Recommendations failed (${res.status})`);
  }
  return res.json();
}

export async function getProfile() {
  const headers = await getHeaders(true);
  const res = await fetch(`${API_URL}/api/v1/profile`, {
    headers,
    credentials: "include",
  });
  if (!res.ok) {
    if (res.status === 404) return null;
    throw new Error("Failed to fetch profile");
  }
  return res.json();
}