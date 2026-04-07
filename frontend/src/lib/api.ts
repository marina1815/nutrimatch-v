const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function loginUser(payload: { email: string; password: string }) {
  const res = await fetch(`${API_URL}/api/auth/login`, {
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
  const res = await fetch(`${API_URL}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error("Register failed");
  return res.json();
}

export async function saveProfile(profile: unknown) {
  const res = await fetch(`${API_URL}/api/profile`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });
  if (!res.ok) throw new Error("Profile save failed");
  return res.json();
}

export async function getRecommendations(profile: unknown) {
  const res = await fetch(`${API_URL}/api/recommendations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });
  if (!res.ok) throw new Error("Recommendations failed");
  return res.json();
}