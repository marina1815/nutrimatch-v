"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getHealthMetrics } from "@/lib/health";

export default function ProfilePage() {
  const [profile, setProfile] = useState<any>(null);
  const metrics = profile ? getHealthMetrics(profile) : null;

  useEffect(() => {
    const saved = localStorage.getItem("nutrimatch-profile");
    if (saved) {
      setProfile(JSON.parse(saved));
    }
  }, []);

  if (!profile) {
    return (
      <main className="nm-page">
        <div className="nm-card">
          <h1 className="nm-title">No profile found</h1>
          <p className="nm-sub">Please complete onboarding first.</p>
          <Link href="/onboarding" className="nm-link-btn nm-link-btn-primary">
            Start onboarding
          </Link>
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
          <div><strong>Likes:</strong> {profile.preferences.likes.join(", ") || "—"}</div>
          <div><strong>Dislikes:</strong> {profile.preferences.dislikes.join(", ") || "—"}</div>
          <div><strong>Meal types:</strong> {profile.preferences.mealTypes.join(", ") || "—"}</div>
          <div><strong>Allergies:</strong> {profile.constraints.allergies.join(", ") || "—"}</div>
          <div><strong>Conditions:</strong> {profile.constraints.conditions.join(", ") || "—"}</div>
          <div><strong>Excluded ingredients:</strong> {profile.constraints.excludedIngredients.join(", ") || "—"}</div>
          {metrics && (
            <div className="nm-card">
              <h2 className="nm-title" style={{ fontSize: "1.4rem" }}>Indicateurs santé</h2>
              <div className="nm-stack">
                <div><strong>IMC :</strong> {metrics.bmi}</div>
                <div><strong>Catégorie IMC :</strong> {metrics.bmiCategory}</div>
                <div><strong>BMR :</strong> {metrics.bmr} kcal/jour</div>
                <div><strong>Besoin calorique estimé :</strong> {metrics.estimatedCalories} kcal/jour</div>
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