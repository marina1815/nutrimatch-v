"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { RecommendationList } from "@/components/results/RecommendationList";
import {
  ApiError,
  getProfile,
  getRecommendationExplanation,
  getRecommendations,
  getRecommendationTrace,
} from "@/lib/api";
import { getCurrentProfileId, setCurrentProfileId } from "@/lib/session";
import {
  MealRecommendation,
  RecommendationExplanation,
  RecommendationTrace,
} from "@/lib/types";
import { getSafeErrorMessage } from "@/lib/ui-errors";

export default function ResultsPage() {
  const [profileId, setProfileId] = useState<string | null>(null);
  const [meals, setMeals] = useState<MealRecommendation[]>([]);
  const [trace, setTrace] = useState<RecommendationTrace | null>(null);
  const [explanationsByMealId, setExplanationsByMealId] = useState<Record<string, RecommendationExplanation>>({});
  const [loadingExplanationMealId, setLoadingExplanationMealId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadRecommendations = async () => {
      try {
        let nextProfileId = getCurrentProfileId();

        if (!nextProfileId) {
          const profile = await getProfile();
          nextProfileId = profile.profileId;
          setCurrentProfileId(nextProfileId);
        }

        const [recommendationResponse, traceResponse] = await Promise.all([
          getRecommendations(nextProfileId),
          getRecommendationTrace(nextProfileId),
        ]);

        if (!cancelled) {
          setProfileId(nextProfileId);
          setMeals(recommendationResponse.meals);
          setTrace(traceResponse);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof ApiError && err.status === 401) {
          setError("Connecte-toi pour consulter tes recommandations.");
        } else {
          setError(getSafeErrorMessage(err, "recommendations.load"));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadRecommendations();

    return () => {
      cancelled = true;
    };
  }, []);

  const handleExplain = async (mealId: string) => {
    if (!profileId || explanationsByMealId[mealId] || loadingExplanationMealId === mealId) {
      return;
    }

    setLoadingExplanationMealId(mealId);
    try {
      const explanation = await getRecommendationExplanation(profileId, mealId);
      setExplanationsByMealId((current) => ({
        ...current,
        [mealId]: explanation,
      }));
    } catch (err) {
      setError(getSafeErrorMessage(err, "recommendations.explain"));
    } finally {
      setLoadingExplanationMealId(null);
    }
  };

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
            <Link href="/onboarding" className="nm-link-btn">Edit profile</Link>
            <Link href="/profile" className="nm-link-btn nm-link-btn-primary">View profile</Link>
          </div>
        </div>

        {trace && (
          <div className="nm-card nm-aux-card">
            <h2 className="nm-title" style={{ fontSize: "1.35rem" }}>Recommendation trace</h2>
            <div className="nm-trace-grid">
              <div className="nm-keyval">
                <span className="nm-muted">Status</span>
                <strong>{trace.status}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">Accepted</span>
                <strong>{String(trace.decisionSummary.accepted ?? 0)}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">Rejected</span>
                <strong>{String(trace.decisionSummary.rejected ?? 0)}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">AI rerank</span>
                <strong>{String(trace.decisionSummary.aiApplied ?? false)}</strong>
              </div>
            </div>
          </div>
        )}

        {loading && (
          <div className="nm-card">
            <p className="nm-sub">Loading recommendations...</p>
          </div>
        )}

        {!loading && error && (
          <div className="nm-card">
            <p className="nm-error">{error}</p>
            <div className="nm-inline-actions">
              <Link href="/login" className="nm-link-btn nm-link-btn-primary">Sign in</Link>
              <Link href="/onboarding" className="nm-link-btn">Edit profile</Link>
            </div>
          </div>
        )}

        {!loading && !error && meals.length === 0 && (
          <div className="nm-card">
            <p className="nm-sub">No safe recommendation is available for this profile right now.</p>
          </div>
        )}

        {!loading && !error && meals.length > 0 && (
          <RecommendationList
            meals={meals}
            explanationsByMealId={explanationsByMealId}
            loadingExplanationMealId={loadingExplanationMealId}
            onExplain={(mealId) => void handleExplain(mealId)}
          />
        )}
      </section>
    </main>
  );
}
