"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { ApiError, getNutritionProfile, getProfile } from "@/lib/api";
import { NutritionProfile, UserProfileResponse } from "@/lib/types";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfileResponse | null>(null);
  const [nutrition, setNutrition] = useState<NutritionProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
          <div><strong>Meal styles:</strong> {profile.preferences.mealStyles.join(", ") || "-"}</div>
          <div><strong>Meal types:</strong> {profile.preferences.mealTypes.join(", ") || "-"}</div>
          <div><strong>Preferred cuisines:</strong> {profile.preferences.preferredCuisines.join(", ") || "-"}</div>
          <div><strong>Excluded cuisines:</strong> {profile.preferences.excludedCuisines.join(", ") || "-"}</div>
          <div><strong>Allergies:</strong> {profile.constraints.allergies.join(", ") || "-"}</div>
          <div><strong>Conditions:</strong> {profile.constraints.conditions.join(", ") || "-"}</div>
          <div><strong>Excluded ingredients:</strong> {profile.constraints.excludedIngredients.join(", ") || "-"}</div>
          {nutrition && (
            <div className="nm-card">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>Health metrics</h2>
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
