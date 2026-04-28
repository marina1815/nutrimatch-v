"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import {
  ApiError,
  beginTotpSetup,
  changePassword,
  confirmTotp,
  disableTotp,
  getMfaStatus,
  getNutritionProfile,
  getProfile,
  MfaStatus,
  registerPasskey,
  setMfaPreference,
  verifyPasskey,
} from "@/lib/api";
import { NutritionProfile, UserProfileResponse } from "@/lib/types";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfileResponse | null>(null);
  const [nutrition, setNutrition] = useState<NutritionProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mfaStatus, setMfaStatus] = useState<MfaStatus | null>(null);
  const [totpSetup, setTotpSetup] = useState<{ secret: string; otpauthUrl: string } | null>(null);
  const [totpCode, setTotpCode] = useState("");
  const [securityMessage, setSecurityMessage] = useState<string | null>(null);
  const [securityError, setSecurityError] = useState<string | null>(null);
  const [passwords, setPasswords] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  });

  async function refreshMfaStatus() {
    try {
      setMfaStatus(await getMfaStatus());
    } catch {
      setMfaStatus(null);
    }
  }

  useEffect(() => {
    let cancelled = false;

    const loadProfile = async () => {
      try {
        const [profileResponse, nutritionResponse] = await Promise.all([
          getProfile(),
          getNutritionProfile(),
        ]);

        if (!cancelled) {
          setProfile(profileResponse);
          setNutrition(nutritionResponse);
          void refreshMfaStatus();
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof ApiError && err.status === 401) {
          setError("Connecte-toi pour consulter ton profil.");
        } else if (err instanceof ApiError && err.status === 404) {
          setError("Aucun profil enregistre pour le moment.");
        } else {
          setError(getSafeErrorMessage(err, "profile.load"));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadProfile();

    return () => {
      cancelled = true;
    };
  }, []);

  const handlePasswordChange = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await changePassword(passwords);
      setPasswords({ currentPassword: "", newPassword: "", confirmPassword: "" });
      setSecurityMessage("Password updated. Other sessions were revoked.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.password"));
    }
  };

  const handleBeginTotp = async () => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      setTotpSetup(await beginTotpSetup());
      setSecurityMessage("Scan the authenticator URI or add the secret manually, then confirm the 6-digit code.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.totp.setup"));
    }
  };

  const handleConfirmTotp = async () => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await confirmTotp(totpCode);
      setTotpCode("");
      setTotpSetup(null);
      await refreshMfaStatus();
      setSecurityMessage("Authenticator MFA is enabled.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.totp.confirm"));
    }
  };

  const handleDisableTotp = async () => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await disableTotp(totpCode);
      setTotpCode("");
      await refreshMfaStatus();
      setSecurityMessage("Authenticator MFA is disabled.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.totp.disable"));
    }
  };

  const handleRegisterPasskey = async () => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await registerPasskey("NutriMatch passkey");
      await refreshMfaStatus();
      setSecurityMessage("Passkey registered.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.passkey.register"));
    }
  };

  const handleSetMfaPreference = async (preferredMethod: "" | "totp" | "passkey") => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await setMfaPreference(preferredMethod);
      await refreshMfaStatus();
      setSecurityMessage(preferredMethod ? "Preferred MFA method updated." : "Preferred MFA method cleared.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.passkey.verify"));
    }
  };

  const handleVerifyPasskey = async () => {
    setSecurityError(null);
    setSecurityMessage(null);
    try {
      await verifyPasskey();
      setSecurityMessage("Passkey verification succeeded.");
    } catch (err) {
      setSecurityError(getSafeErrorMessage(err, "profile.security.passkey.verify"));
    }
  };

  if (loading) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">Loading profile</h1>
          <p className="nm-sub">Fetching your saved nutrition profile.</p>
        </div>
      </main>
    );
  }

  if (!profile) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">No profile found</h1>
          <p className="nm-sub">{error || "Please complete onboarding first."}</p>
          <div className="nm-inline-actions">
            <Link href="/onboarding" className="nm-link-btn nm-link-btn-primary">
              Start onboarding
            </Link>
            <Link href="/login" className="nm-link-btn">
              Sign in
            </Link>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="nm-page">
      <div className="nm-card">
        <span className="nm-logo">NutriMatch</span>
        <h1 className="nm-title">{profile.personal.fullName}</h1>
        <p className="nm-sub">Your nutrition profile summary</p>

        <div className="nm-stack">
          <div><strong>Age:</strong> {profile.personal.age}</div>
          <div><strong>Sex:</strong> {profile.personal.sex}</div>
          <div><strong>Weight:</strong> {profile.personal.weight} kg</div>
          <div><strong>Height:</strong> {profile.personal.height} cm</div>
          <div><strong>Activity:</strong> {profile.lifestyle.activityLevel}</div>
          <div><strong>Goal:</strong> {profile.lifestyle.goal}</div>
          <div><strong>Max ready time:</strong> {profile.lifestyle.maxReadyTime} min</div>
          <div><strong>Likes:</strong> {profile.preferences.likes.join(", ") || "-"}</div>
          <div><strong>Dislikes:</strong> {profile.preferences.dislikes.join(", ") || "-"}</div>
          <div><strong>Meal types:</strong> {profile.preferences.mealTypes.join(", ") || "-"}</div>
          <div><strong>Preferred cuisines:</strong> {profile.preferences.preferredCuisines.join(", ") || "-"}</div>
          <div><strong>Excluded cuisines:</strong> {profile.preferences.excludedCuisines.join(", ") || "-"}</div>
          <div><strong>Allergies:</strong> {profile.constraints.allergies.join(", ") || "-"}</div>
          <div><strong>Conditions:</strong> {profile.constraints.conditions.join(", ") || "-"}</div>
          <div><strong>Excluded ingredients:</strong> {profile.constraints.excludedIngredients.join(", ") || "-"}</div>
          {profile.constraints.takesMedication && profile.constraints.medicationsRedacted && (
            <div>
              <strong>Medications:</strong> hidden in this summary for safety
            </div>
          )}
          {nutrition && (
            <div className="nm-card">
              <h2 className="nm-title nm-section-title">Health metrics</h2>
              <div className="nm-stack">
                <div><strong>BMI:</strong> {nutrition.bmi}</div>
                <div><strong>BMI category:</strong> {nutrition.bmiCategory}</div>
                <div><strong>BMR:</strong> {nutrition.bmr} kcal/day</div>
                <div><strong>Estimated calories:</strong> {nutrition.estimatedCalories} kcal/day</div>
                <div><strong>Target calories:</strong> {nutrition.targetCalories} kcal/day</div>
                <div><strong>Protein target:</strong> {nutrition.targetProteinGrams} g/day</div>
              </div>
            </div>
          )}
          <div className="nm-card">
            <h2 className="nm-title nm-section-title">Security</h2>
            <p className="nm-sub">
              Password changes revoke other sessions. MFA can use an authenticator app or a passkey.
            </p>
            {securityMessage && <p className="nm-sub">{securityMessage}</p>}
            {securityError && <p className="nm-error">{securityError}</p>}

            <form className="nm-stack" onSubmit={handlePasswordChange}>
              <input
                className="nm-input"
                type="password"
                autoComplete="current-password"
                placeholder="Current password"
                value={passwords.currentPassword}
                onChange={(event) => setPasswords((value) => ({ ...value, currentPassword: event.target.value }))}
                required
              />
              <input
                className="nm-input"
                type="password"
                autoComplete="new-password"
                placeholder="New password"
                minLength={12}
                value={passwords.newPassword}
                onChange={(event) => setPasswords((value) => ({ ...value, newPassword: event.target.value }))}
                required
              />
              <input
                className="nm-input"
                type="password"
                autoComplete="new-password"
                placeholder="Confirm new password"
                minLength={12}
                value={passwords.confirmPassword}
                onChange={(event) => setPasswords((value) => ({ ...value, confirmPassword: event.target.value }))}
                required
              />
              <button className="nm-link-btn nm-link-btn-primary" type="submit">
                Update password
              </button>
            </form>

            <div className="nm-stack">
              <div><strong>Authenticator:</strong> {mfaStatus?.totpEnabled ? "enabled" : "disabled"}</div>
              <div><strong>Passkeys:</strong> {mfaStatus?.passkeyCount ?? 0}</div>
              <div><strong>Preferred MFA:</strong> {mfaStatus?.effectiveMethod || "none"}</div>
              {totpSetup && (
                <div className="nm-stack">
                  <div><strong>Secret:</strong> {totpSetup.secret}</div>
                  <div className="nm-break-all"><strong>OTP URI:</strong> {totpSetup.otpauthUrl}</div>
                </div>
              )}
              <input
                className="nm-input"
                inputMode="numeric"
                pattern="[0-9]{6}"
                placeholder="6-digit authenticator code"
                value={totpCode}
                onChange={(event) => setTotpCode(event.target.value.replace(/\D/g, "").slice(0, 6))}
              />
              <div className="nm-inline-actions">
                <button className="nm-link-btn" type="button" onClick={handleBeginTotp}>
                  Setup authenticator
                </button>
                <button className="nm-link-btn nm-link-btn-primary" type="button" onClick={handleConfirmTotp}>
                  Confirm authenticator
                </button>
                {mfaStatus?.totpEnabled && (
                  <button className="nm-link-btn" type="button" onClick={handleDisableTotp}>
                    Disable authenticator
                  </button>
                )}
              </div>
              <div className="nm-inline-actions">
                <button className="nm-link-btn" type="button" onClick={handleRegisterPasskey}>
                  Add passkey
                </button>
                {mfaStatus?.passkeyEnabled && (
                  <button className="nm-link-btn nm-link-btn-primary" type="button" onClick={handleVerifyPasskey}>
                    Verify passkey
                  </button>
                )}
              </div>
              <div className="nm-inline-actions">
                {mfaStatus?.totpEnabled && (
                  <button className="nm-link-btn" type="button" onClick={() => void handleSetMfaPreference("totp")}>
                    Prefer authenticator
                  </button>
                )}
                {mfaStatus?.passkeyEnabled && (
                  <button className="nm-link-btn" type="button" onClick={() => void handleSetMfaPreference("passkey")}>
                    Prefer passkey
                  </button>
                )}
                {mfaStatus?.stepUpAvailable && (
                  <button className="nm-link-btn" type="button" onClick={() => void handleSetMfaPreference("")}>
                    Auto-select MFA
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        <div className="nm-inline-actions">
          <Link href="/onboarding" className="nm-link-btn">
            Edit profile
          </Link>
          <Link href="/results" className="nm-link-btn nm-link-btn-primary">
            See results
          </Link>
        </div>
      </div>
    </main>
  );
}
