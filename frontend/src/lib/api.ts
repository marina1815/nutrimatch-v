import "client-only";

import {
  CurrentSession,
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

type ApiMeta = {
  requestId?: string;
  timestamp?: string;
};

type ApiSuccessEnvelope<T> = {
  data: T;
  meta?: ApiMeta;
};

type ApiErrorEnvelope = {
  error?: {
    code?: string;
    message?: string;
  };
  meta?: ApiMeta;
};

export class ApiError extends Error {
  status: number;
  code?: string;
  requestId?: string;

  constructor(message: string, status: number, code?: string, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  try {
    const payload = (await response.json()) as ApiErrorEnvelope | { error?: string };
    if ("error" in payload && typeof payload.error === "object" && payload.error) {
      return payload.error.message || fallback;
    }
    if ("error" in payload && typeof payload.error === "string") {
      return payload.error || fallback;
    }
    return fallback;
  } catch {
    return fallback;
  }
}

async function readApiError(response: Response, fallback: string): Promise<ApiError> {
  try {
    const payload = (await response.json()) as ApiErrorEnvelope | { error?: string; request_id?: string };
    if ("error" in payload && typeof payload.error === "object" && payload.error) {
      const requestId = "meta" in payload && payload.meta ? payload.meta.requestId : undefined;
      return new ApiError(
        payload.error.message || fallback,
        response.status,
        payload.error.code,
        requestId,
      );
    }
    if ("error" in payload && typeof payload.error === "string") {
      return new ApiError(
        payload.error || fallback,
        response.status,
        undefined,
        "request_id" in payload ? payload.request_id : undefined,
      );
    }
  } catch {
    return new ApiError(fallback, response.status);
  }

  return new ApiError(fallback, response.status);
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
    token: payload.csrf_token || "",
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
    throw await readApiError(response, "API request failed");
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = (await response.json()) as ApiSuccessEnvelope<T> | T;
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data;
  }
  return payload as T;
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

export async function getProfile(options: { includeSensitive?: boolean } = {}) {
  const params = new URLSearchParams();
  if (options.includeSensitive) {
    params.set("includeSensitive", "true");
  }

  const suffix = params.size > 0 ? `?${params.toString()}` : "";
  return apiRequest<UserProfileResponse>(
    `/api/v1/profile${suffix}`,
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function getCurrentSession() {
  return apiRequest<CurrentSession>(
    "/api/v1/auth/whoami",
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
