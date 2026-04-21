import {
  MealRecommendation,
  RecommendationExplanation,
  RecommendationTrace,
  NutritionProfile,
  UserProfile,
  UserProfileResponse,
} from "@/lib/types";
import {
  clearAccessToken,
  clearClientSession,
  getAccessToken,
  setAccessToken,
} from "@/lib/session";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const DEFAULT_CSRF_HEADER = "X-CSRF-Token";

type CsrfTokenResponse = {
  csrf_token: string;
  header_name?: string;
};

type AuthResponse = {
  access_token: string;
  expires_at: string;
};

type RecommendationResponse = {
  runId: string;
  profileId: string;
  meals: MealRecommendation[];
};

type RequestOptions = {
  auth?: boolean;
  csrf?: boolean;
  retryOnUnauthorized?: boolean;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function getCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const prefix = `${name}=`;
  const cookie = document.cookie
    .split("; ")
    .find((item) => item.startsWith(prefix));

  if (!cookie) {
    return null;
  }

  return decodeURIComponent(cookie.slice(prefix.length));
}

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  try {
    const payload = (await response.json()) as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

async function ensureCsrfToken(): Promise<{ token: string; headerName: string }> {
  const response = await fetch(`${API_URL}/api/v1/auth/csrf`, {
    method: "GET",
    credentials: "include",
  });

  if (!response.ok) {
    throw new ApiError(await readErrorMessage(response, "Failed to issue CSRF token"), response.status);
  }

  const payload = (await response.json()) as CsrfTokenResponse;

  return {
    token: payload.csrf_token || getCookie("nutrimatch_csrf") || "",
    headerName: payload.header_name || DEFAULT_CSRF_HEADER,
  };
}

async function refreshAccessToken(): Promise<string | null> {
  try {
    const csrf = await ensureCsrfToken();
    const response = await fetch(`${API_URL}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: {
        [csrf.headerName]: csrf.token,
      },
    });

    if (!response.ok) {
      clearAccessToken();
      return null;
    }

    const payload = (await response.json()) as AuthResponse;
    setAccessToken(payload.access_token);
    return payload.access_token;
  } catch {
    clearAccessToken();
    return null;
  }
}

async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(init.headers);

  if (options.csrf) {
    const csrf = await ensureCsrfToken();
    headers.set(csrf.headerName, csrf.token);
  }

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (options.auth) {
    let accessToken = getAccessToken();
    if (!accessToken) {
      accessToken = await refreshAccessToken();
    }

    if (!accessToken) {
      throw new ApiError("Authentication required", 401);
    }

    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers,
    credentials: "include",
  });

  if (
    response.status === 401 &&
    options.auth &&
    options.retryOnUnauthorized !== false
  ) {
    const refreshedToken = await refreshAccessToken();
    if (refreshedToken) {
      return apiRequest<T>(path, init, {
        ...options,
        retryOnUnauthorized: false,
      });
    }
  }

  if (!response.ok) {
    throw new ApiError(
      await readErrorMessage(response, "API request failed"),
      response.status,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export async function loginUser(payload: { email: string; password: string }) {
  const response = await apiRequest<AuthResponse>(
    "/api/v1/auth/login",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    { csrf: true },
  );

  setAccessToken(response.access_token);
  return response;
}

export async function registerUser(payload: {
  name: string;
  email: string;
  password: string;
}) {
  const response = await apiRequest<AuthResponse>(
    "/api/v1/auth/register",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    { csrf: true },
  );

  setAccessToken(response.access_token);
  return response;
}

export async function logoutUser() {
  await apiRequest<void>(
    "/api/v1/auth/logout",
    {
      method: "POST",
    },
    { csrf: true, retryOnUnauthorized: false },
  );

  clearClientSession();
}

export async function submitProfile(profile: UserProfile) {
  return apiRequest<{ profileId: string }>(
    "/api/v1/profile",
    {
      method: "POST",
      body: JSON.stringify(profile),
    },
    { auth: true },
  );
}

export async function getProfile() {
  return apiRequest<UserProfileResponse>(
    "/api/v1/profile",
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function getNutritionProfile() {
  return apiRequest<NutritionProfile>(
    "/api/v1/profile/nutrition",
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function getRecommendations(profileId: string) {
  return apiRequest<RecommendationResponse>(
    `/api/v1/recommendations/${profileId}`,
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function getRecommendationTrace(profileId: string) {
  return apiRequest<RecommendationTrace>(
    `/api/v1/recommendations/${profileId}/trace`,
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function getRecommendationExplanation(profileId: string, mealId: string) {
  const params = new URLSearchParams({ mealId });
  return apiRequest<RecommendationExplanation>(
    `/api/v1/recommendations/${profileId}/explanation?${params.toString()}`,
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function suggestIngredients(query: string, limit = 5) {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
  });

  const response = await apiRequest<{ items: string[] }>(
    `/api/v1/profile/ingredients/suggest?${params.toString()}`,
    {
      method: "GET",
    },
    { auth: true },
  );

  return response.items;
}
