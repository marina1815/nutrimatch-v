"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { getHealthMetrics } from "@/lib/health";
import { UserProfile } from "@/lib/types";
import { PersonalInfoStep } from "@/components/forms/PersonalInfoStep";
import { LifestyleStep } from "@/components/forms/LifeStyleStep";
import { PreferencesStep } from "@/components/forms/PreferencesStep";
import { ConstraintsStep } from "@/components/forms/ConstraintsStep";
import { validateProfile, ProfileErrors } from "@/lib/validation";
import { saveProfile } from "@/lib/api";
import { Button } from "@/components/ui/Button";

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [errors, setErrors] = useState<ProfileErrors>({});
  const [isSaving, setIsSaving] = useState(false);
  // Snapshot du profil original avant modification — pour annuler
  const originalProfile = useRef<UserProfile | null>(null);

  useEffect(() => {
    const saved = localStorage.getItem("nutrimatch-profile");
    if (saved) {
      const parsed = JSON.parse(saved) as UserProfile;
      setProfile(parsed);
      originalProfile.current = parsed;
    }
  }, []);

  const metrics = profile ? getHealthMetrics(profile) : null;

  const handleStartEdit = () => {
    // Prendre un snapshot avant de modifier
    originalProfile.current = profile ? JSON.parse(JSON.stringify(profile)) : null;
    setIsEditing(true);
  };

  const handleCancel = () => {
    // Restaurer le snapshot original — toutes les modifications sont annulées
    if (originalProfile.current) {
      setProfile(originalProfile.current);
    }
    setIsEditing(false);
    setErrors({});
  };

  const handleSave = async () => {
    if (!profile) return;

    const validationErrors = validateProfile(profile);
    if (Object.keys(validationErrors).length > 0) {
      setErrors(validationErrors);
      alert("Veuillez corriger les erreurs avant d'enregistrer.");
      return;
    }

    setIsSaving(true);
    try {
      await saveProfile(profile);
      localStorage.setItem("nutrimatch-profile", JSON.stringify(profile));
      originalProfile.current = JSON.parse(JSON.stringify(profile));
      setIsEditing(false);
      setErrors({});
      alert("Profil mis à jour avec succès !");
    } catch (err) {
      console.error(err);
      alert("Erreur lors de l'enregistrement du profil.");
    } finally {
      setIsSaving(false);
    }
  };

  if (!profile) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">Aucun profil trouvé</h1>
          <p className="nm-sub">Veuillez d&apos;abord compléter l&apos;onboarding.</p>
          <Link href="/onboarding" className="nm-link-btn nm-link-btn-primary">
            Commencer l&apos;onboarding
          </Link>
        </div>
      </main>
    );
  }

  if (isEditing) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">Modifier mon profil</h1>
          <p className="nm-sub">
            Vos informations actuelles sont pré-remplies — modifiez uniquement ce que vous souhaitez changer.
          </p>

          <div className="nm-stack" style={{ gap: "2.5rem" }}>
            <section className="nm-stack">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>1. Informations personnelles</h2>
              <PersonalInfoStep
                data={profile}
                setData={setProfile as React.Dispatch<React.SetStateAction<UserProfile>>}
                errors={errors.personal}
              />
            </section>

            <section className="nm-stack">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>2. Mode de vie</h2>
              <LifestyleStep
                data={profile}
                setData={setProfile as React.Dispatch<React.SetStateAction<UserProfile>>}
                errors={errors.lifestyle}
              />
            </section>

            <section className="nm-stack">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>3. Préférences</h2>
              <PreferencesStep
                data={profile}
                setData={setProfile as React.Dispatch<React.SetStateAction<UserProfile>>}
                errors={errors.preferences}
              />
            </section>

            <section className="nm-stack">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>4. Santé &amp; contraintes</h2>
              <ConstraintsStep
                data={profile}
                setData={setProfile as React.Dispatch<React.SetStateAction<UserProfile>>}
                errors={errors.constraints}
              />
            </section>
          </div>

          <div className="nm-inline-actions" style={{ marginTop: "2rem" }}>
            <Button variant="secondary" onClick={handleCancel}>
              Annuler
            </Button>
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Enregistrement..." : "Enregistrer les modifications"}
            </Button>
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
        <p className="nm-sub">Récapitulatif de votre profil nutritionnel</p>

        <div className="nm-stack nm-summary-grid">
          <div className="nm-summary-item"><strong>Âge:</strong> {profile.personal?.age || "—"} ans</div>
          <div className="nm-summary-item"><strong>Sexe:</strong> {profile.personal?.sex === "male" ? "Homme" : "Femme"}</div>
          <div className="nm-summary-item"><strong>Poids:</strong> {profile.personal?.weight || "—"} kg</div>
          <div className="nm-summary-item"><strong>Taille:</strong> {profile.personal?.height || "—"} cm</div>
          <div className="nm-summary-item"><strong>Activité:</strong> {profile.lifestyle?.activityLevel || "—"}</div>
          <div className="nm-summary-item"><strong>Objectif:</strong> {profile.lifestyle?.goal || "—"}</div>
          <div className="nm-summary-item"><strong>Régimes:</strong> {profile.preferences?.diets?.join(", ") || "—"}</div>
          <div className="nm-summary-item"><strong>Cuisines:</strong> {profile.preferences?.cuisines?.join(", ") || "—"}</div>
          <div className="nm-summary-item"><strong>Allergies:</strong> {profile.constraints?.allergies?.join(", ") || "—"}</div>
        </div>

        {metrics && (
          <div className="nm-card nm-metrics-card">
            <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>Indicateurs santé</h2>
            <div className="nm-grid">
              <div><strong>IMC:</strong> {metrics.bmi} ({metrics.bmiCategory})</div>
              <div><strong>BMR:</strong> {metrics.bmr} kcal/j</div>
              <div><strong>Besoin estimé:</strong> {metrics.estimatedCalories} kcal/j</div>
            </div>
          </div>
        )}

        <div className="nm-inline-actions">
          <Button variant="secondary" onClick={handleStartEdit}>
            Modifier le profil
          </Button>
          <Link href="/results" className="nm-link-btn nm-link-btn-primary">
            Voir les recommandations
          </Link>
        </div>
      </div>
    </main>
  );
}