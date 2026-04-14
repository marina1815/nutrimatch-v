const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

// Handle CSRF persistence
async function ensureCSRF(): Promise<{ name: string; token: string }> {
  if (typeof window === "undefined") return { name: "X-CSRF-Token", token: "" };
  
  const saved = sessionStorage.getItem("nutrimatch-csrf");
  const savedHeader = sessionStorage.getItem("nutrimatch-csrf-header") || "X-CSRF-Token";
  
  if (saved) {
    return { name: savedHeader, token: saved };
  }

  // Fetch new token with credentials to get the cookie
  const res = await fetch(`${API_URL}/api/v1/auth/csrf`, {
    credentials: "include"
  });
  if (!res.ok) throw new Error("Failed to initialize security context (CSRF)");
  
  const data = await res.json();
  sessionStorage.setItem("nutrimatch-csrf", data.csrf_token);
  sessionStorage.setItem("nutrimatch-csrf-header", data.header_name);
  
  return { name: data.header_name, token: data.csrf_token };
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
  return res.json();
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
  return res.json();
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
  if (!res.ok) throw new Error("Failed to fetch recommendations");
  return res.json();
}