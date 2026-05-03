import "client-only";

import {
  AuthSession,
  CatalogOption,
  CurrentSession,
  MealChoiceResponse,
  NutritionProfile,
  ProfileTaxonomy,
  RecommendationExplanation,
  RecommendationResponse,
  RecommendationTrace,
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

type MfaChallengeResponse = {
  mfa_required: true;
  challenge_id: string;
  preferred_method: "totp" | "passkey";
  allowed_methods: Array<"totp" | "passkey">;
  expires_at: string;
};

function isMfaChallengeResponse(value: AuthResponse | MfaChallengeResponse): value is MfaChallengeResponse {
  return "mfa_required" in value && value.mfa_required === true;
}

export type MfaStatus = {
  totpEnabled: boolean;
  passkeyEnabled: boolean;
  passkeyCount: number;
  stepUpAvailable: boolean;
  preferredMethod: "" | "totp" | "passkey";
  effectiveMethod: "" | "totp" | "passkey";
};

export type TotpSetup = {
  secret: string;
  otpauthUrl: string;
};

type PasskeyOptionsResponse = {
  challengeId: string;
  options: unknown;
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

let refreshInFlight: Promise<string | null> | null = null;

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

async function ensureCsrfToken(accessToken?: string | null): Promise<{ token: string; headerName: string }> {
  const headers = new Headers();
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`${API_URL}/api/v1/auth/csrf`, {
    method: "GET",
    credentials: "include",
    headers,
  });

  if (!response.ok) {
    throw new ApiError(await readErrorMessage(response, "Failed to issue CSRF token"), response.status);
  }

  const payload = (await response.json()) as ApiSuccessEnvelope<CsrfTokenResponse> | CsrfTokenResponse;
  const data = "data" in payload ? payload.data : payload;

  return {
    token: data.csrf_token || "",
    headerName: data.header_name || DEFAULT_CSRF_HEADER,
  };
}

async function doRefreshAccessToken(): Promise<string | null> {
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

    const payload = (await response.json()) as ApiSuccessEnvelope<AuthResponse> | AuthResponse;
    const data = "data" in payload ? payload.data : payload;
    setAccessToken(data.access_token);
    return data.access_token;
  } catch {
    clearAccessToken();
    return null;
  }
}

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = doRefreshAccessToken().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  let accessToken: string | null = null;

  if (options.auth) {
    accessToken = getAccessToken();
    if (!accessToken) {
      accessToken = await refreshAccessToken();
    }

    if (!accessToken) {
      throw new ApiError("Authentication required", 401);
    }

    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  if (options.csrf) {
    const csrf = await ensureCsrfToken(accessToken);
    headers.set(csrf.headerName, csrf.token);
  }

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
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
  const response = await apiRequest<AuthResponse | MfaChallengeResponse>(
    "/api/v1/auth/login",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    { csrf: true },
  );

  if (!isMfaChallengeResponse(response)) {
    setAccessToken(response.access_token);
  }
  return response;
}

export async function completeTotpLogin(payload: { challengeId: string; code: string }) {
  const response = await apiRequest<AuthResponse>(
    "/api/v1/auth/mfa/login/totp",
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

export async function changePassword(payload: {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}) {
  return apiRequest<void>(
    "/api/v1/auth/password/change",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    { auth: true, csrf: true },
  );
}

export async function getMfaStatus() {
  return apiRequest<MfaStatus>(
    "/api/v1/auth/mfa/status",
    {
      method: "GET",
    },
    { auth: true },
  );
}

export async function setMfaPreference(preferredMethod: "" | "totp" | "passkey") {
  return apiRequest<void>(
    "/api/v1/auth/mfa/preference",
    {
      method: "POST",
      body: JSON.stringify({ preferredMethod }),
    },
    { auth: true, csrf: true },
  );
}

export async function beginTotpSetup() {
  return apiRequest<TotpSetup>(
    "/api/v1/auth/mfa/totp/setup",
    {
      method: "POST",
    },
    { auth: true, csrf: true },
  );
}

export async function confirmTotp(code: string) {
  return apiRequest<void>(
    "/api/v1/auth/mfa/totp/confirm",
    {
      method: "POST",
      body: JSON.stringify({ code }),
    },
    { auth: true, csrf: true },
  );
}

export async function disableTotp(code: string) {
  return apiRequest<void>(
    "/api/v1/auth/mfa/totp/disable",
    {
      method: "POST",
      body: JSON.stringify({ code }),
    },
    { auth: true, csrf: true },
  );
}

export async function listSessions() {
  const response = await apiRequest<{ sessions: AuthSession[] }>(
    "/api/v1/auth/sessions",
    {
      method: "GET",
    },
    { auth: true },
  );
  return response.sessions;
}

export async function revokeSession(sessionId: string) {
  return apiRequest<void>(
    `/api/v1/auth/sessions/${encodeURIComponent(sessionId)}`,
    {
      method: "DELETE",
    },
    { auth: true, csrf: true },
  );
}

export async function beginPasskeyRegistration() {
  return apiRequest<PasskeyOptionsResponse>(
    "/api/v1/auth/mfa/passkeys/registration/options",
    {
      method: "POST",
    },
    { auth: true, csrf: true },
  );
}

export async function finishPasskeyRegistration(challengeId: string, displayName: string, credential: PublicKeyCredential) {
  const params = new URLSearchParams({
    challengeId,
    displayName,
  });
  return apiRequest<void>(
    `/api/v1/auth/mfa/passkeys/registration/finish?${params.toString()}`,
    {
      method: "POST",
      body: JSON.stringify(publicKeyCredentialToJSON(credential)),
    },
    { auth: true, csrf: true },
  );
}

export async function beginPasskeyAuthentication() {
  return apiRequest<PasskeyOptionsResponse>(
    "/api/v1/auth/mfa/passkeys/authentication/options",
    {
      method: "POST",
    },
    { auth: true, csrf: true },
  );
}

export async function finishPasskeyAuthentication(challengeId: string, credential: PublicKeyCredential) {
  const params = new URLSearchParams({ challengeId });
  return apiRequest<void>(
    `/api/v1/auth/mfa/passkeys/authentication/finish?${params.toString()}`,
    {
      method: "POST",
      body: JSON.stringify(publicKeyCredentialToJSON(credential)),
    },
    { auth: true, csrf: true },
  );
}

export async function beginLoginPasskey(challengeId: string) {
  return apiRequest<PasskeyOptionsResponse>(
    "/api/v1/auth/mfa/login/passkeys/options",
    {
      method: "POST",
      body: JSON.stringify({ challengeId }),
    },
    { csrf: true },
  );
}

export async function finishLoginPasskey(challengeId: string, passkeyChallengeId: string, credential: PublicKeyCredential) {
  const params = new URLSearchParams({
    challengeId,
    passkeyChallengeId,
  });
  const response = await apiRequest<AuthResponse>(
    `/api/v1/auth/mfa/login/passkeys/finish?${params.toString()}`,
    {
      method: "POST",
      body: JSON.stringify(publicKeyCredentialToJSON(credential)),
    },
    { csrf: true },
  );

  setAccessToken(response.access_token);
  return response;
}

export async function submitProfile(profile: UserProfile) {
  return apiRequest<{ profileId: string }>(
    "/api/v1/profile",
    {
      method: "POST",
      body: JSON.stringify(profile),
    },
    { auth: true, csrf: true },
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

export async function getProfileTaxonomy() {
  return apiRequest<ProfileTaxonomy>(
    "/api/v1/profile/taxonomy",
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
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 15_000);

  try {
    return await apiRequest<RecommendationExplanation>(
      `/api/v1/recommendations/${profileId}/explanation?${params.toString()}`,
      {
        method: "GET",
        signal: controller.signal,
      },
      { auth: true },
    );
  } finally {
    window.clearTimeout(timeout);
  }
}

export async function refreshRecommendationExplanations(profileId: string) {
  return apiRequest<RecommendationResponse>(
    `/api/v1/recommendations/${profileId}/explanations/refresh`,
    {
      method: "POST",
    },
    { auth: true, csrf: true },
  );
}

export async function chooseRecommendationMeal(profileId: string, mealId: string) {
  return apiRequest<MealChoiceResponse>(
    `/api/v1/recommendations/${profileId}/meals/${encodeURIComponent(mealId)}/choose`,
    {
      method: "POST",
    },
    { auth: true, csrf: true },
  );
}

export async function suggestIngredients(query: string, limit = 5) {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
  });

  const response = await apiRequest<{ items: CatalogOption[] }>(
    `/api/v1/profile/ingredients/suggest?${params.toString()}`,
    {
      method: "GET",
    },
    { auth: true },
  );

  return response.items;
}

function publicKeyCredentialToJSON(credential: PublicKeyCredential) {
  const response = credential.response as AuthenticatorAttestationResponse | AuthenticatorAssertionResponse;
  const payload: Record<string, unknown> = {
    id: credential.id,
    rawId: arrayBufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {},
  };

  if ("attestationObject" in response) {
    payload.response = {
      clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
      attestationObject: arrayBufferToBase64Url(response.attestationObject),
    };
  } else {
    payload.response = {
      clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
      authenticatorData: arrayBufferToBase64Url(response.authenticatorData),
      signature: arrayBufferToBase64Url(response.signature),
      userHandle: response.userHandle ? arrayBufferToBase64Url(response.userHandle) : null,
    };
  }
  return payload;
}

function arrayBufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return window.btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}
