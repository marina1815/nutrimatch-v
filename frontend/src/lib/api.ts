const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function loginUser(payload: { email: string; password: string }) {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error("Login failed");
  return res.json();
}

export async function registerUser(payload: {
  name: string;
  email: string;
  password: string;
}) {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error("Register failed");
  return res.json();
}

export async function submitProfile(profile: unknown) {
  const res = await fetch(`${API_URL}/api/v1/profile`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });

  if (!res.ok) throw new Error("Failed to submit profile");
  return res.json();
}

export async function getRecommendations(profileId: string) {
  const res = await fetch(`${API_URL}/api/v1/recommendations/${profileId}`);
  if (!res.ok) throw new Error("Failed to fetch recommendations");
  return res.json();
}