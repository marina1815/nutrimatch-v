"use client";

import { useRouter } from "next/navigation";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { PersonalInfoStep } from "@/components/forms/PersonalInfoStep";
import { LifestyleStep } from "@/components/forms/LifeStyleStep";
import { PreferencesStep } from "@/components/forms/PreferencesStep";
import { ConstraintsStep } from "@/components/forms/ConstraintsStep";
import { useProfileForm } from "@/hooks/useProfileForm";
import { saveProfile } from "@/lib/api";
import { UserProfile } from "@/lib/types";

const steps = ["Infos personnelles", "Mode de vie", "Préférences", "Santé & contraintes"];

/**
 * Transform the frontend profile (which stores numbers as strings in form state)
 * into the exact shape the backend API expects (numbers as numbers).
 */
function buildApiPayload(data: UserProfile) {
  return {
    personal: {
      fullName: data.personal.fullName,
      age: Number(data.personal.age),
      sex: data.personal.sex,
      weight: Number(data.personal.weight),
      height: Number(data.personal.height),
      profession: data.personal.profession,
      city: data.personal.city,
    },
    lifestyle: {
      activityLevel: data.lifestyle.activityLevel,
      lifestyleType: data.lifestyle.lifestyleType,
      goal: data.lifestyle.goal,
    },
    preferences: {
      likes: data.preferences.likes,
      dislikes: data.preferences.dislikes,
      mealStyles: data.preferences.mealStyles,
      diets: data.preferences.diets,
      cuisines: data.preferences.cuisines,
      mealsPerDay: Number(data.preferences.mealsPerDay),
    },
    constraints: {
      allergies: data.constraints.allergies,
      conditions: data.constraints.conditions,
      excludedIngredients: data.constraints.excludedIngredients,
      hasChronicDisease: data.constraints.hasChronicDisease,
      chronicDiseases: data.constraints.chronicDiseases,
      takesMedication: data.constraints.takesMedication,
      medications: data.constraints.medications,
    },
  };
}

export default function OnboardingPage() {
  const router = useRouter();
  const {
    step, data, setData, errors,
    isSubmitting, setIsSubmitting, setSubmitError,
    next, back,
  } = useProfileForm();

  const handleNext = async () => {
    if (step < 3) {
      next();
      return;
    }

    try {
      setIsSubmitting(true);
      setSubmitError(null);

      const payload = buildApiPayload(data);
      const response = await saveProfile(payload);
      console.log("Profile saved:", response);

      router.push("/results");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to save profile. Please check your connection.";
      console.error("Submission error:", message);
      setSubmitError(message);
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
            Étape {step + 1} sur 4 — {steps[step]}
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

        <div className="nm-actions">
          <Button variant="secondary" onClick={back} disabled={step === 0}>
            Retour
          </Button>

          <Button onClick={handleNext} disabled={isSubmitting}>
            {isSubmitting ? "Envoi en cours..." : step === 3 ? "Voir les recommandations" : "Continuer"}
          </Button>
        </div>
      </Card>
    </main>
  );
}