"use client";

import { FormEvent, useEffect, useState } from "react";
import {
  ApiError,
  beginPasskeyAuthentication,
  beginPasskeyRegistration,
  beginTotpSetup,
  changePassword,
  confirmTotp,
  disableTotp,
  finishPasskeyAuthentication,
  finishPasskeyRegistration,
  getMfaStatus,
  MfaStatus,
  setMfaPreference,
  TotpSetup,
} from "@/lib/api";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export function SecurityPanel() {
  const [mfa, setMfa] = useState<MfaStatus | null>(null);
  const [totpSetup, setTotpSetup] = useState<TotpSetup | null>(null);
  const [totpCode, setTotpCode] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [passkeyName, setPasskeyName] = useState("Clé d'accès NutriMatch");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [passwords, setPasswords] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  });

  const refreshSecurityState = async () => {
    setMfa(await getMfaStatus());
  };

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const mfaStatus = await getMfaStatus();
        if (!cancelled) {
          setMfa(mfaStatus);
        }
      } catch (err) {
        if (!cancelled) {
          setError(getSafeErrorMessage(err, "profile.security.load"));
        }
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const runAction = async (label: string, action: () => Promise<void>, success: string) => {
    setBusy(label);
    setError(null);
    setMessage(null);
    try {
      await action();
      await refreshSecurityState();
      setMessage(success);
    } catch (err) {
      setError(err instanceof ApiError ? getSafeErrorMessage(err, "profile.security.action") : getSafeErrorMessage(err, "profile.security.action"));
    } finally {
      setBusy(null);
    }
  };

  const handlePasswordChange = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await runAction("password", async () => {
      await changePassword(passwords);
      setPasswords({ currentPassword: "", newPassword: "", confirmPassword: "" });
    }, "Mot de passe mis à jour.");
  };

  const handleBeginTotp = async () => {
    await runAction("totp-setup", async () => {
      const setup = await beginTotpSetup();
      setTotpSetup(setup);
    }, "Secret authenticator généré. Ajoute-le dans ton application, puis confirme le code.");
  };

  const handleConfirmTotp = async () => {
    await runAction("totp-confirm", async () => {
      await confirmTotp(totpCode);
      setTotpSetup(null);
      setTotpCode("");
    }, "Authenticator activé.");
  };

  const handleDisableTotp = async () => {
    await runAction("totp-disable", async () => {
      await disableTotp(disableCode);
      setDisableCode("");
    }, "Authenticator désactivé.");
  };

  const handleRegisterPasskey = async () => {
    await runAction("passkey-register", async () => {
      ensureWebAuthnAvailable();
      const begin = await beginPasskeyRegistration();
      const credential = await navigator.credentials.create(normalizeCredentialCreationOptions(begin.options));
      if (!credential) {
        throw new ApiError("Enregistrement de la clé d'accès annulé", 400, "PASSKEY_CANCELLED");
      }
      await finishPasskeyRegistration(begin.challengeId, passkeyName, credential as PublicKeyCredential);
    }, "Clé d'accès ajoutée.");
  };

  const handleVerifyPasskey = async () => {
    await runAction("passkey-verify", async () => {
      ensureWebAuthnAvailable();
      const begin = await beginPasskeyAuthentication();
      const credential = await navigator.credentials.get(normalizeCredentialRequestOptions(begin.options));
      if (!credential) {
        throw new ApiError("Vérification de la clé d'accès annulée", 400, "PASSKEY_CANCELLED");
      }
      await finishPasskeyAuthentication(begin.challengeId, credential as PublicKeyCredential);
    }, "Clé d'accès vérifiée.");
  };

  const handlePreferenceChange = async (method: "" | "totp" | "passkey") => {
    await runAction("mfa-preference", async () => {
      await setMfaPreference(method);
    }, "Préférence MFA mise à jour.");
  };

  return (
    <div className="nm-card">
      <h2 className="nm-title nm-section-title">Sécurité</h2>
      <p className="nm-sub">
        Une seule session locale est autorisée. La MFA reste optionnelle, mais sera demandée après le mot de passe si tu l&apos;actives.
      </p>
      {message && <p className="nm-sub">{message}</p>}
      {error && <p className="nm-error">{error}</p>}

      <form className="nm-stack" onSubmit={(event) => void handlePasswordChange(event)}>
        <h3>Mot de passe</h3>
        <input className="nm-input" type="password" autoComplete="current-password" placeholder="Mot de passe actuel" value={passwords.currentPassword} onChange={(event) => setPasswords((value) => ({ ...value, currentPassword: event.target.value }))} required />
        <input className="nm-input" type="password" autoComplete="new-password" placeholder="Nouveau mot de passe" minLength={12} value={passwords.newPassword} onChange={(event) => setPasswords((value) => ({ ...value, newPassword: event.target.value }))} required />
        <input className="nm-input" type="password" autoComplete="new-password" placeholder="Confirmation" minLength={12} value={passwords.confirmPassword} onChange={(event) => setPasswords((value) => ({ ...value, confirmPassword: event.target.value }))} required />
        <button className="nm-link-btn nm-link-btn-primary" type="submit" disabled={busy === "password"}>Mettre à jour</button>
      </form>

      <div className="nm-stack">
        <h3>Authenticator TOTP</h3>
        <p className="nm-muted">État: {mfa?.totpEnabled ? "activé" : "inactif"}</p>
        {!mfa?.totpEnabled && (
          <button className="nm-link-btn" type="button" onClick={() => void handleBeginTotp()} disabled={busy === "totp-setup"}>Configurer authenticator</button>
        )}
        {totpSetup && (
          <div className="nm-explain-box">
            <p className="nm-muted">Secret: <strong>{totpSetup.secret}</strong></p>
            <p className="nm-muted">{totpSetup.otpauthUrl}</p>
            <input className="nm-input" inputMode="numeric" maxLength={6} placeholder="Code 6 chiffres" value={totpCode} onChange={(event) => setTotpCode(event.target.value.replace(/\D/g, "").slice(0, 6))} />
            <button className="nm-link-btn nm-link-btn-primary" type="button" onClick={() => void handleConfirmTotp()} disabled={busy === "totp-confirm" || totpCode.length !== 6}>Confirmer TOTP</button>
          </div>
        )}
        {mfa?.totpEnabled && (
          <div className="nm-stack">
            <input className="nm-input" inputMode="numeric" maxLength={6} placeholder="Code actuel pour désactiver" value={disableCode} onChange={(event) => setDisableCode(event.target.value.replace(/\D/g, "").slice(0, 6))} />
            <button className="nm-link-btn" type="button" onClick={() => void handleDisableTotp()} disabled={busy === "totp-disable" || disableCode.length !== 6}>Désactiver TOTP</button>
          </div>
        )}
      </div>

      <div className="nm-stack">
        <h3>Clés d&apos;accès WebAuthn</h3>
        <p className="nm-muted">Clés actives: {mfa?.passkeyCount ?? 0}</p>
        <input className="nm-input" value={passkeyName} maxLength={80} onChange={(event) => setPasskeyName(event.target.value)} placeholder="Nom de la clé d'accès" />
        <div className="nm-inline-actions">
          <button className="nm-link-btn" type="button" onClick={() => void handleRegisterPasskey()} disabled={busy === "passkey-register"}>Ajouter une clé</button>
          {mfa?.passkeyEnabled && <button className="nm-link-btn" type="button" onClick={() => void handleVerifyPasskey()} disabled={busy === "passkey-verify"}>Tester la clé</button>}
        </div>
      </div>

      {mfa?.stepUpAvailable && (
        <div className="nm-stack">
          <h3>Préférence MFA</h3>
          <select className="nm-input" value={mfa.preferredMethod} onChange={(event) => void handlePreferenceChange(event.target.value as "" | "totp" | "passkey")}>
            <option value="">Automatique</option>
            {mfa.totpEnabled && <option value="totp">Authenticator</option>}
            {mfa.passkeyEnabled && <option value="passkey">Clé d&apos;accès</option>}
          </select>
          <p className="nm-muted">Méthode effective: {mfa.effectiveMethod || "aucune"}</p>
        </div>
      )}
    </div>
  );
}

function ensureWebAuthnAvailable() {
  if (typeof window === "undefined" || !window.PublicKeyCredential || !navigator.credentials) {
    throw new ApiError("WebAuthn n'est pas disponible dans ce navigateur", 400, "WEBAUTHN_UNAVAILABLE");
  }
}

type RawCredentialDescriptor = Omit<PublicKeyCredentialDescriptor, "id"> & {
  id: string;
};

type RawCredentialCreationOptions = Omit<
  PublicKeyCredentialCreationOptions,
  "challenge" | "user" | "excludeCredentials"
> & {
  challenge: string;
  user: Omit<PublicKeyCredentialUserEntity, "id"> & { id: string };
  excludeCredentials?: RawCredentialDescriptor[];
};

type RawCredentialRequestOptions = Omit<
  PublicKeyCredentialRequestOptions,
  "challenge" | "allowCredentials"
> & {
  challenge: string;
  allowCredentials?: RawCredentialDescriptor[];
};

function normalizeCredentialCreationOptions(raw: unknown): CredentialCreationOptions {
  const options = unwrapPublicKeyOptions(raw) as RawCredentialCreationOptions;
  return {
    publicKey: {
      ...options,
      challenge: base64UrlToArrayBuffer(options.challenge),
      user: {
        ...options.user,
        id: base64UrlToArrayBuffer(options.user.id),
      },
      excludeCredentials: options.excludeCredentials?.map((credential) => ({
        ...credential,
        id: base64UrlToArrayBuffer(credential.id),
      })),
    },
  };
}

function normalizeCredentialRequestOptions(raw: unknown): CredentialRequestOptions {
  const options = unwrapPublicKeyOptions(raw) as RawCredentialRequestOptions;
  return {
    publicKey: {
      ...options,
      challenge: base64UrlToArrayBuffer(options.challenge),
      allowCredentials: options.allowCredentials?.map((credential) => ({
        ...credential,
        id: base64UrlToArrayBuffer(credential.id),
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
