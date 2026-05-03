"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { SecurityPanel } from "@/components/security/SecurityPanel";
import { ApiError, getNutritionProfile, getProfile } from "@/lib/api";
import { clearClientSession } from "@/lib/session";
import {
  labelActivity,
  labelAllergies,
  labelBmiCategory,
  labelConditions,
  labelGoal,
  labelIngredients,
  labelLifestyle,
  labelSex,
} from "@/lib/display-labels";
import { NutritionProfile, UserProfileResponse } from "@/lib/types";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function ProfilePage() {
  const router = useRouter();
  const [profile, setProfile] = useState<UserProfileResponse | null>(null);
  const [nutrition, setNutrition] = useState<NutritionProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [errorStatus, setErrorStatus] = useState<number | null>(null);

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
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof ApiError && err.status === 401) {
          clearClientSession();
          setErrorStatus(401);
          setError("Connecte-toi pour consulter ton profil.");
          router.replace("/login");
        } else if (err instanceof ApiError && err.status === 404) {
          setErrorStatus(404);
          setError("Aucun profil enregistré pour le moment.");
        } else {
          setErrorStatus(null);
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
  }, [router]);

  if (loading) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">Chargement du profil</h1>
          <p className="nm-sub">Récupération de ton profil nutritionnel sauvegardé.</p>
        </div>
      </main>
    );
  }

  if (!profile) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">Aucun profil trouvé</h1>
          <p className="nm-sub">{error || "Complète d'abord ton onboarding."}</p>
          <div className="nm-inline-actions">
            {errorStatus !== 401 && (
              <Link href="/onboarding" className="nm-link-btn nm-link-btn-primary">
                Commencer l&apos;onboarding
              </Link>
            )}
            <Link href="/login" className="nm-link-btn">
              Se connecter
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
        <p className="nm-sub">Résumé de ton profil nutritionnel</p>

        <div className="nm-stack">
          <div><strong>Âge:</strong> {profile.personal.age}</div>
          <div><strong>Sexe:</strong> {labelSex(profile.personal.sex)}</div>
          <div><strong>Poids:</strong> {profile.personal.weight} kg</div>
          <div><strong>Taille:</strong> {profile.personal.height} cm</div>
          <div><strong>Activité:</strong> {labelActivity(profile.lifestyle.activityLevel)}</div>
          <div><strong>Mode de vie:</strong> {labelLifestyle(profile.lifestyle.lifestyleType)}</div>
          <div><strong>Objectif:</strong> {labelGoal(profile.lifestyle.goal)}</div>
          <div><strong>Aliments aimés:</strong> {labelIngredients(profile.preferences.likes)}</div>
          <div><strong>Aliments moins appréciés:</strong> {labelIngredients(profile.preferences.dislikes)}</div>
          <div><strong>Sensibilités alimentaires:</strong> {labelAllergies(profile.constraints.allergies)}</div>
          <div><strong>Maladies:</strong> {labelConditions(profile.constraints.conditions)}</div>
          <div><strong>Ingrédients exclus:</strong> {labelIngredients(profile.constraints.excludedIngredients)}</div>
          {profile.constraints.takesMedication && profile.constraints.medicationsRedacted && (
            <div>
              <strong>Médicaments:</strong> masqués dans ce résumé pour protéger tes données
            </div>
          )}
          {nutrition && (
            <div className="nm-card">
              <h2 className="nm-title nm-section-title">Indicateurs nutritionnels</h2>
              <div className="nm-stack">
                <div><strong>IMC:</strong> {nutrition.bmi}</div>
                <div><strong>Catégorie IMC:</strong> {labelBmiCategory(nutrition.bmiCategory)}</div>
                <div><strong>Métabolisme basal:</strong> {nutrition.bmr} kcal/jour</div>
                <div><strong>Calories estimées:</strong> {nutrition.estimatedCalories} kcal/jour</div>
                <div><strong>Objectif calorique:</strong> {nutrition.targetCalories} kcal/jour</div>
                <div><strong>Objectif protéines:</strong> {nutrition.targetProteinGrams} g/jour</div>
              </div>
            </div>
          )}
          <SecurityPanel />
        </div>

        <div className="nm-inline-actions">
          <Link href="/onboarding" className="nm-link-btn">
            Modifier le profil
          </Link>
          <Link href="/results" className="nm-link-btn nm-link-btn-primary">
            Voir les résultats
          </Link>
        </div>
      </div>
    </main>
  );
}
