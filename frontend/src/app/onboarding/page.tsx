"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { ConstraintsStep } from "@/components/forms/ConstraintsStep";
import { LifestyleStep } from "@/components/forms/LifeStyleStep";
import { PersonalInfoStep } from "@/components/forms/PersonalInfoStep";
import { PreferencesStep } from "@/components/forms/PreferencesStep";
import { ApiError, getProfileTaxonomy, submitProfile } from "@/lib/api";
import { sanitizeProfile } from "@/lib/profile-normalization";
import { clearClientSession } from "@/lib/session";
import { ProfileTaxonomy } from "@/lib/types";
import { getSafeErrorMessage } from "@/lib/ui-errors";
import { useProfileForm } from "@/hooks/useProfileForm";

const steps = ["Infos personnelles", "Mode de vie", "Préférences", "Santé & contraintes"];

export default function OnboardingPage() {
  const router = useRouter();
  const {
    step,
    data,
    setData,
    errors,
    loadingSavedProfile,
    loadSavedProfileError,
    authRequired,
    next,
    back,
    reset,
  } = useProfileForm();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [taxonomy, setTaxonomy] = useState<ProfileTaxonomy | null>(null);
  const [taxonomyError, setTaxonomyError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadTaxonomy = async () => {
      try {
        const response = await getProfileTaxonomy();
        if (!cancelled) {
          setTaxonomy(response);
          setTaxonomyError(null);
        }
      } catch {
        if (!cancelled) {
          setTaxonomyError("Catalogue partiellement indisponible: options de secours chargees.");
        }
      }
    };

    void loadTaxonomy();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (authRequired) {
      clearClientSession();
      router.replace("/login");
    }
  }, [authRequired, router]);

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
      await submitProfile(sanitizeProfile(data));
      reset();
      router.push("/results");
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setSubmitError("Session expirée. Connecte-toi de nouveau pour enregistrer ton profil.");
        router.push("/login");
      } else {
        setSubmitError(getSafeErrorMessage(error, "profile.submit"));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loadingSavedProfile) {
    return (
      <main className="nm-page">
        <Card>
          <div className="nm-header">
            <span className="nm-logo">NutriMatch</span>
            <h1 className="nm-title">Préparation du formulaire</h1>
            <p className="nm-sub">On vérifie si un profil existe déjà pour éviter de tout ressaisir.</p>
          </div>
        </Card>
      </main>
    );
  }

  if (authRequired) {
    return (
      <main className="nm-page">
        <Card>
          <div className="nm-header">
            <span className="nm-logo">NutriMatch</span>
            <h1 className="nm-title">Connexion requise</h1>
            <p className="nm-sub">Connecte-toi avant de créer ou modifier ton profil nutritionnel.</p>
          </div>
          <div className="nm-actions">
            <Button onClick={() => router.push("/login")}>Se connecter</Button>
          </div>
        </Card>
      </main>
    );
  }

  return (
    <main className="nm-page">
      <Card>
        <div className="nm-header">
          <span className="nm-logo">NutriMatch</span>
          <h1 className="nm-title">Construis ton profil nutritionnel</h1>
          <p className="nm-sub">
            Étape {step + 1} sur 4 - {steps[step]}
          </p>
          {loadSavedProfileError && <p className="nm-error">{loadSavedProfileError}</p>}
          {taxonomyError && <p className="nm-error">{taxonomyError}</p>}
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
            <ConstraintsStep
              data={data}
              setData={setData}
              errors={errors.constraints}
              taxonomy={taxonomy}
            />
          )}
        </div>

        {submitError && <p className="nm-error">{submitError}</p>}

        <div className="nm-actions nm-form-actions">
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
