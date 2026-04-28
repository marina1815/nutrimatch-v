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

type TotpSetupResponse = {
  secret: string;
  otpauthUrl: string;
};

type PasskeyOptionsResponse = {
  challengeId: string;
  options: unknown;
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

  const payload = (await response.json()) as ApiSuccessEnvelope<CsrfTokenResponse> | CsrfTokenResponse;
  const data = "data" in payload ? payload.data : payload;

  return {
    token: data.csrf_token || "",
    headerName: data.header_name || DEFAULT_CSRF_HEADER,
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

    const payload = (await response.json()) as ApiSuccessEnvelope<AuthResponse> | AuthResponse;
    const data = "data" in payload ? payload.data : payload;
    setAccessToken(data.access_token);
    return data.access_token;
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
  return apiRequest<TotpSetupResponse>(
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

export async function registerPasskey(displayName = "NutriMatch passkey") {
  if (!window.PublicKeyCredential) {
    throw new ApiError("Passkeys are not supported by this browser", 400, "PASSKEY_UNSUPPORTED");
  }

  const begin = await apiRequest<PasskeyOptionsResponse>(
    "/api/v1/auth/mfa/passkeys/registration/options",
    { method: "POST" },
    { auth: true, csrf: true },
  );
  const credential = await navigator.credentials.create(normalizeCredentialCreationOptions(begin.options));
  if (!credential) {
    throw new ApiError("Passkey registration was cancelled", 400, "PASSKEY_CANCELLED");
  }

  const params = new URLSearchParams({
    challengeId: begin.challengeId,
    displayName,
  });
  return apiRequest<void>(
    `/api/v1/auth/mfa/passkeys/registration/finish?${params.toString()}`,
    {
      method: "POST",
      body: JSON.stringify(publicKeyCredentialToJSON(credential as PublicKeyCredential)),
    },
    { auth: true, csrf: true },
  );
}

export async function verifyPasskey() {
  if (!window.PublicKeyCredential) {
    throw new ApiError("Passkeys are not supported by this browser", 400, "PASSKEY_UNSUPPORTED");
  }

  const begin = await apiRequest<PasskeyOptionsResponse>(
    "/api/v1/auth/mfa/passkeys/authentication/options",
    { method: "POST" },
    { auth: true, csrf: true },
  );
  const credential = await navigator.credentials.get(normalizeCredentialRequestOptions(begin.options));
  if (!credential) {
    throw new ApiError("Passkey verification was cancelled", 400, "PASSKEY_CANCELLED");
  }

  const params = new URLSearchParams({ challengeId: begin.challengeId });
  return apiRequest<void>(
    `/api/v1/auth/mfa/passkeys/authentication/finish?${params.toString()}`,
    {
      method: "POST",
      body: JSON.stringify(publicKeyCredentialToJSON(credential as PublicKeyCredential)),
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

function normalizeCredentialCreationOptions(raw: unknown): CredentialCreationOptions {
  const options = unwrapPublicKeyOptions(raw) as PublicKeyCredentialCreationOptions;
  return {
    publicKey: {
      ...options,
      challenge: base64UrlToArrayBuffer(options.challenge as unknown as string),
      user: {
        ...options.user,
        id: base64UrlToArrayBuffer(options.user.id as unknown as string),
      },
      excludeCredentials: options.excludeCredentials?.map((credential) => ({
        ...credential,
        id: base64UrlToArrayBuffer(credential.id as unknown as string),
      })),
    },
  };
}

function normalizeCredentialRequestOptions(raw: unknown): CredentialRequestOptions {
  const options = unwrapPublicKeyOptions(raw) as PublicKeyCredentialRequestOptions;
  return {
    publicKey: {
      ...options,
      challenge: base64UrlToArrayBuffer(options.challenge as unknown as string),
      allowCredentials: options.allowCredentials?.map((credential) => ({
        ...credential,
        id: base64UrlToArrayBuffer(credential.id as unknown as string),
      })),
    },
  };
}

function unwrapPublicKeyOptions(raw: unknown): unknown {
  if (raw && typeof raw === "object" && "publicKey" in raw) {
    return (raw as { publicKey: unknown }).publicKey;
  }
  return raw;
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

function base64UrlToArrayBuffer(value: string): ArrayBuffer {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(normalized.length + ((4 - (normalized.length % 4)) % 4), "=");
  const binary = window.atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

function arrayBufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return window.btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}
