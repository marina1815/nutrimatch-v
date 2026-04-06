const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function submitProfile(profile: unknown) {
  const res = await fetch(`${API_URL}/api/profile`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });

  if (!res.ok) throw new Error("Failed to submit profile");
  return res.json();
}

export async function getRecommendations(profileId: string) {
  const res = await fetch(`${API_URL}/api/recommendations/${profileId}`);
  if (!res.ok) throw new Error("Failed to fetch recommendations");
  return res.json();
}