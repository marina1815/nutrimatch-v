"use client";

import { useRouter } from "next/navigation";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { PersonalInfoStep } from "@/components/forms/PersonalInfoStep";
import { LifestyleStep } from "@/components/forms/LifeStyleStep";
import { PreferencesStep } from "@/components/forms/PreferencesStep";
import { ConstraintsStep } from "@/components/forms/ConstraintsStep";
import { useProfileForm } from "@/hooks/useProfileForm";

const steps = ["Infos personnelles", "Mode de vie", "Préférences", "Santé & contraintes"];

export default function OnboardingPage() {
  const router = useRouter();
  const { step, data, setData, errors, next, back } = useProfileForm();

  const handleNext = () => {
    if (step < 3) {
      next();
      return;
    }
    router.push("/results");
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

          <Button onClick={handleNext}>
            {step === 3 ? "Voir les recommandations" : "Continuer"}
          </Button>
        </div>
      </Card>
    </main>
  );
}