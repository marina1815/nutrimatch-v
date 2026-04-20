"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { RecommendationList } from "@/components/results/RecommendationList";
import { getProfile, getRecommendations } from "@/lib/api";
import { MealRecommendation } from "@/lib/types";

export default function ResultsPage() {
  const [meals, setMeals] = useState<MealRecommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadResults() {
      try {
        setLoading(true);
        setError(null);
        
        // 1. Get the profile first (to get the ID)
        // Note: backend returns "profileId" field (not "id")
        const profile = await getProfile();
        if (!profile || !profile.profileId) {
          setError("No profile found. Please complete the onboarding first.");
          return;
        }

        // 2. Get recommendations based on profile ID
        const response = await getRecommendations(profile.profileId);
        setMeals(response.meals || []);
      } catch (err: unknown) {
        console.error("Failed to load recommendations:", err);
        const message = err instanceof Error ? err.message : "An error occurred while fetching your personal recommendations.";
        setError(message);
      } finally {
        setLoading(false);
      }
    }

    loadResults();
  }, []);

  return (
    <main className="nm-page">
      <section className="nm-results-shell">
        <div className="nm-header-row">
          <div>
            <span className="nm-logo">NutriMatch</span>
            <h1 className="nm-title">Your meal recommendations</h1>
            <p className="nm-sub">
              Based on your profile, preferences, lifestyle and dietary constraints.
            </p>
          </div>

          <div className="nm-inline-actions">
            <Link href="/profile" className="nm-link-btn">Modifier le profil</Link>
            <Link href="/profile" className="nm-link-btn nm-link-btn-primary">View profile</Link>
          </div>
        </div>

        {loading ? (
          <div className="nm-loading-state">
            <div className="nm-spinner"></div>
            <p>Analyzing your nutritional needs and finding the best matches...</p>
          </div>
        ) : error ? (
          <div className="nm-error-state">
            <p>{error}</p>
            <Link href="/onboarding" className="nm-link-btn">Start Onboarding</Link>
          </div>
        ) : (
          <RecommendationList meals={meals} />
        )}
      </section>
    </main>
  );
}