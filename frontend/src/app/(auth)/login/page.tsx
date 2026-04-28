"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  ApiError,
  beginLoginPasskey,
  completeTotpLogin,
  finishLoginPasskey,
  getCurrentSession,
  loginUser,
} from "@/lib/api";
import { setCurrentProfileId } from "@/lib/session";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function LoginPage() {
  const router = useRouter();
  const [form, setForm] = useState({ email: "", password: "" });
  const [mfa, setMfa] = useState<{
    challengeId: string;
    preferredMethod: "totp" | "passkey";
    allowedMethods: Array<"totp" | "passkey">;
  } | null>(null);
  const [totpCode, setTotpCode] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [loading, setLoading] = useState(false);

  const validate = () => {
    const nextErrors: typeof errors = {};
    if (!form.email.includes("@")) nextErrors.email = "Enter a valid email address";
    if (form.password.length < 12) nextErrors.password = "Password must be at least 12 characters";
    return nextErrors;
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const nextErrors = validate();
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }

    setLoading(true);
    setErrors({});

    try {
      const result = await loginUser(form);
      if ("mfa_required" in result) {
        setMfa({
          challengeId: result.challenge_id,
          preferredMethod: result.preferred_method,
          allowedMethods: result.allowed_methods,
        });
        setForm((current) => ({ ...current, password: "" }));
        return;
      }
      await redirectAfterLogin();
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors({ form: getSafeErrorMessage(error, "auth.login") });
      } else {
        setErrors({ form: getSafeErrorMessage(error, "auth.login") });
      }
    } finally {
      setLoading(false);
    }
  };

  const redirectAfterLogin = async () => {
    try {
      const session = await getCurrentSession();
      if (session.profileId) {
        setCurrentProfileId(session.profileId);
      }
      router.push(session.hasProfile ? "/results" : "/onboarding");
    } catch {
      router.push("/onboarding");
    }
  };

  const handleTotpSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!mfa) return;
    setLoading(true);
    setErrors({});
    try {
      await completeTotpLogin({ challengeId: mfa.challengeId, code: totpCode });
      await redirectAfterLogin();
    } catch (error) {
      setErrors({ form: getSafeErrorMessage(error, "auth.login") });
    } finally {
      setLoading(false);
    }
  };

  const handlePasskeyLogin = async () => {
    if (!mfa) return;
    setLoading(true);
    setErrors({});
    try {
      const begin = await beginLoginPasskey(mfa.challengeId);
      const credential = await navigator.credentials.get(normalizeCredentialRequestOptions(begin.options));
      if (!credential) {
        throw new ApiError("Passkey verification was cancelled", 400, "PASSKEY_CANCELLED");
      }
      await finishLoginPasskey(mfa.challengeId, begin.challengeId, credential as PublicKeyCredential);
      await redirectAfterLogin();
    } catch (error) {
      setErrors({ form: getSafeErrorMessage(error, "auth.login") });
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="page">
      <div className="card">
        <Link href="/" className="logo">NutriMatch</Link>

        <h1 className="title">Welcome back</h1>
        <p className="sub">Sign in to access your nutrition profile</p>

        {!mfa && (
        <form onSubmit={(event) => void handleSubmit(event)} className="form" noValidate>
          <div className="field">
            <label className="label" htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              maxLength={254}
              className={`input ${errors.email ? "input-error" : ""}`}
              value={form.email}
              onChange={(event) => {
                setForm({ ...form, email: event.target.value });
                setErrors({ ...errors, email: undefined, form: undefined });
              }}
            />
            {errors.email && <span className="error">{errors.email}</span>}
          </div>

          <div className="field">
            <div className="label-row">
              <label className="label" htmlFor="password">Password</label>
              <span className="forgot">Forgot password?</span>
            </div>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              placeholder="........"
              maxLength={128}
              className={`input ${errors.password ? "input-error" : ""}`}
              value={form.password}
              onChange={(event) => {
                setForm({ ...form, password: event.target.value });
                setErrors({ ...errors, password: undefined, form: undefined });
              }}
            />
            {errors.password && <span className="error">{errors.password}</span>}
          </div>

          {errors.form && <span className="error">{errors.form}</span>}

          <button type="submit" className="btn" disabled={loading}>
            {loading ? <span className="spinner" /> : "Sign in"}
          </button>
        </form>
        )}

        {mfa && (
          <form onSubmit={(event) => void handleTotpSubmit(event)} className="form" noValidate>
            <p className="sub">
              Multi-factor verification is required because this account has MFA enabled.
            </p>
            {mfa.allowedMethods.includes("totp") && (
              <div className="field">
                <label className="label" htmlFor="totp">Authenticator code</label>
                <input
                  id="totp"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  className="input"
                  value={totpCode}
                  onChange={(event) => setTotpCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
                  placeholder="123456"
                />
              </div>
            )}

            {errors.form && <span className="error">{errors.form}</span>}

            {mfa.allowedMethods.includes("totp") && (
              <button type="submit" className="btn" disabled={loading || totpCode.length !== 6}>
                {loading ? <span className="spinner" /> : "Verify authenticator"}
              </button>
            )}

            {mfa.allowedMethods.includes("passkey") && (
              <button type="button" className="btn" disabled={loading} onClick={() => void handlePasskeyLogin()}>
                {loading && mfa.preferredMethod === "passkey" ? <span className="spinner" /> : "Verify passkey"}
              </button>
            )}

            <button
              type="button"
              className="switch-link"
              onClick={() => {
                setMfa(null);
                setTotpCode("");
                setErrors({});
              }}
            >
              Use another account
            </button>
          </form>
        )}

        <p className="switch">
          No account yet?{" "}
          <Link href="/register" className="switch-link">Create one</Link>
        </p>
      </div>
    </main>
  );
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
