"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { ConstraintsStep } from "@/components/forms/ConstraintsStep";
import { LifestyleStep } from "@/components/forms/LifeStyleStep";
import { PersonalInfoStep } from "@/components/forms/PersonalInfoStep";
import { PreferencesStep } from "@/components/forms/PreferencesStep";
import { ApiError, submitProfile } from "@/lib/api";
import { sanitizeProfile } from "@/lib/profile-normalization";
import { setCurrentProfileId } from "@/lib/session";
import { getSafeErrorMessage } from "@/lib/ui-errors";
import { useProfileForm } from "@/hooks/useProfileForm";

const steps = ["Infos personnelles", "Mode de vie", "Preferences", "Sante & contraintes"];

export default function OnboardingPage() {
  const router = useRouter();
  const { step, data, setData, errors, next, back, reset } = useProfileForm();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const handleNext = async () => {
    const isValid = next();
    if (!isValid) {
      return;
    }

    if (step < 3) {
      return;
    }

    setSubmitError(null);
    setIsSubmitting(true);

    try {
      const response = await submitProfile(sanitizeProfile(data));
      setCurrentProfileId(response.profileId);
      reset();
      router.push("/results");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setSubmitError("Session expiree. Connecte-toi de nouveau pour enregistrer ton profil.");
        router.push("/login");
      } else {
        setSubmitError(getSafeErrorMessage(error, "profile.submit"));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="nm-page">
      <Card>
        <div className="nm-header">
          <span className="nm-logo">NutriMatch</span>
          <h1 className="nm-title">Construis ton profil nutritionnel</h1>
          <p className="nm-sub">
            Etape {step + 1} sur 4 - {steps[step]}
          </p>
        </div>

        <div className="nm-progress">
          {steps.map((label, index) => (
            <div
              key={label}
              className={`nm-progress-step ${index <= step ? "active" : ""}`}
            >
              {label}
            </div>
          ))}
        </div>

        <div className="nm-content">
          {step === 0 && (
            <PersonalInfoStep data={data} setData={setData} errors={errors.personal} />
          )}

          {step === 1 && (
            <LifestyleStep data={data} setData={setData} errors={errors.lifestyle} />
          )}

          {step === 2 && (
            <PreferencesStep data={data} setData={setData} errors={errors.preferences} />
          )}

          {step === 3 && (
            <ConstraintsStep data={data} setData={setData} errors={errors.constraints} />
          )}
        </div>

        {submitError && <p className="nm-error">{submitError}</p>}

        <div className="nm-actions">
          <Button variant="secondary" onClick={back} disabled={step === 0 || isSubmitting}>
            Retour
          </Button>

          <Button onClick={() => void handleNext()} disabled={isSubmitting}>
            {isSubmitting
              ? "Enregistrement..."
              : step === 3
                ? "Voir les recommandations"
                : "Continuer"}
          </Button>
        </div>
      </Card>
    </main>
  );
}
