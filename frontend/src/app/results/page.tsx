"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { RecommendationList } from "@/components/results/RecommendationList";
import {
  ApiError,
  chooseRecommendationMeal,
  getProfile,
  getRecommendations,
  refreshRecommendationExplanations,
} from "@/lib/api";
import { clearClientSession } from "@/lib/session";
import { MealChoiceResponse, MealRecommendation, RecommendationResponse } from "@/lib/types";
import { sanitizeDisplayText } from "@/lib/text-sanitization";
import { getSafeErrorMessage } from "@/lib/ui-errors";
import { isAIRetryableReason, labelAIReason, labelAIStatus, labelSelectionMode } from "@/lib/display-labels";

const LOADER_STEPS = [
  "Analyse du profil nutritionnel",
  "Application des allergies, exclusions et maladies",
  "Sélection de recettes sûres dans le catalogue local",
  "Génération ou récupération des explications IA",
];

function secondsUntil(date?: string): number {
  if (!date) {
    return 0;
  }
  const ms = new Date(date).getTime() - Date.now();
  return Math.max(0, Math.floor(ms / 1000));
}

function formatCountdown(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export default function ResultsPage() {
  const router = useRouter();
  const [profileId, setProfileId] = useState<string | null>(null);
  const [response, setResponse] = useState<RecommendationResponse | null>(null);
  const [meals, setMeals] = useState<MealRecommendation[]>([]);
  const [selectedChoice, setSelectedChoice] = useState<MealChoiceResponse | null>(null);
  const [choosingMealId, setChoosingMealId] = useState<string | null>(null);
  const [loaderStep, setLoaderStep] = useState(0);
  const [countdown, setCountdown] = useState(0);
  const [loading, setLoading] = useState(true);
  const [refreshingExplanation, setRefreshingExplanation] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [explanationError, setExplanationError] = useState<string | null>(null);
  const [requiresAuth, setRequiresAuth] = useState(false);

  useEffect(() => {
    if (!loading) {
      return;
    }
    const timer = window.setInterval(() => {
      setLoaderStep((current) => Math.min(current + 1, LOADER_STEPS.length - 1));
    }, 1200);
    return () => window.clearInterval(timer);
  }, [loading]);

  useEffect(() => {
    let cancelled = false;

    const loadRecommendations = async () => {
      try {
        const profile = await getProfile();
        const nextProfileId = profile.profileId;

        const recommendationResponse = await getRecommendations(nextProfileId);

        if (!cancelled) {
          setProfileId(nextProfileId);
          setResponse(recommendationResponse);
          setSelectedChoice(recommendationResponse.activeChoice || null);
          setMeals(recommendationResponse.activeChoice ? [] : recommendationResponse.meals);
          setExplanationError(null);
          setCountdown(secondsUntil(recommendationResponse.nextRefreshAt));
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof ApiError && err.status === 401) {
          clearClientSession();
          setRequiresAuth(true);
          setError("Connecte-toi pour consulter tes recommandations.");
          router.replace("/login");
        } else if (err instanceof ApiError && err.status === 404) {
          setError("Complète ton profil avant de consulter tes recommandations.");
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
  }, [router]);

  useEffect(() => {
    if (!response?.nextRefreshAt) {
      return;
    }
    const timer = window.setInterval(() => {
      setCountdown(secondsUntil(response.nextRefreshAt));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [response?.nextRefreshAt]);

  const handleChoose = async (mealId: string) => {
    if (!profileId || choosingMealId) {
      return;
    }
    setChoosingMealId(mealId);
    try {
      const choice = await chooseRecommendationMeal(profileId, mealId);
      setSelectedChoice(choice);
      setMeals([]);
      setResponse((current) => current ? { ...current, activeChoice: choice, meals: [] } : current);
    } catch (err) {
      setError(getSafeErrorMessage(err, "recommendations.choose"));
    } finally {
      setChoosingMealId(null);
    }
  };

  const handleRetryExplanation = async () => {
    if (!profileId || refreshingExplanation) {
      return;
    }
    setRefreshingExplanation(true);
    setExplanationError(null);
    try {
      const refreshed = await refreshRecommendationExplanations(profileId);
      setResponse(refreshed);
      setSelectedChoice(refreshed.activeChoice || null);
      setMeals(refreshed.activeChoice ? [] : refreshed.meals);
      setCountdown(secondsUntil(refreshed.nextRefreshAt));
    } catch (err) {
      setExplanationError(getSafeErrorMessage(err, "recommendations.load"));
    } finally {
      setRefreshingExplanation(false);
    }
  };

  const selectedSubstitutions = selectedChoice?.substitutions ?? [];
  const aiReason = response?.aiSkippedReason || response?.aiOutputIgnoredReason || "";
  const canRetryExplanation = isAIRetryableReason(aiReason);

  return (
    <main className="nm-page">
      <section className="nm-results-shell">
        <div className="nm-header-row">
          <div>
            <span className="nm-logo">NutriMatch</span>
            <h1 className="nm-title">Tes recommandations du jour</h1>
            <p className="nm-sub">
              20 recettes sûres issues du catalogue local, renouvelées toutes les 24 heures.
            </p>
          </div>
        </div>

        {response && (
          <div className="nm-card nm-aux-card">
            <h2 className="nm-title nm-section-title-compact">Prochaine sélection</h2>
            <div className="nm-trace-grid">
              <div className="nm-keyval">
                <span className="nm-muted">Recettes affichées</span>
                <strong>{selectedChoice ? 1 : meals.length}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">Renouvellement</span>
                <strong>{formatCountdown(countdown)}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">Mode</span>
                <strong>{labelSelectionMode(response.selectionMode)}</strong>
              </div>
              <div className="nm-keyval">
                <span className="nm-muted">IA</span>
                <strong>{labelAIStatus(response)}</strong>
              </div>
            </div>
            {response.aiSkippedReason && (
              <p className="nm-muted">{sanitizeDisplayText(labelAIReason(response.aiSkippedReason))}</p>
            )}
            {response.aiOutputIgnoredReason && (
              <p className="nm-muted">Sortie IA ignorée: {sanitizeDisplayText(labelAIReason(response.aiOutputIgnoredReason))}</p>
            )}
          </div>
        )}

        {selectedChoice && (
          <div className="nm-card nm-aux-card">
            <h2 className="nm-title nm-section-title-compact">
              Recette choisie: {sanitizeDisplayText(selectedChoice.meal.title)}
            </h2>
            {selectedChoice.preparationGuide ? (
              <p className="nm-reason">{sanitizeDisplayText(selectedChoice.preparationGuide)}</p>
            ) : (
              <p className="nm-muted">
                {labelAIReason(selectedChoice.aiSkippedReason || selectedChoice.aiOutputIgnoredReason) || "Guide IA indisponible pour cette recette."}
              </p>
            )}
            {selectedSubstitutions.length > 0 && (
              <div className="nm-ingredients">
                <strong>Substitutions compatibles</strong>
                <div className="nm-stack">
                  {selectedSubstitutions.map((substitution) => (
                    <p className="nm-muted" key={`${substitution.from}-${substitution.to}`}>
                      {sanitizeDisplayText(substitution.from)} vers {sanitizeDisplayText(substitution.to)}: {sanitizeDisplayText(substitution.reason)}
                    </p>
                  ))}
                </div>
              </div>
            )}
            <p className="nm-muted">
              Cette recette sera exclue des nouvelles suggestions jusqu&apos;au {new Date(selectedChoice.excludedUntil).toLocaleString("fr-FR")}.
            </p>
          </div>
        )}

        {loading && (
          <div className="nm-card">
            <p className="nm-sub">{LOADER_STEPS[loaderStep]}...</p>
          </div>
        )}

        {!loading && error && (
          <div className="nm-card">
            <p className="nm-error">{error}</p>
            <div className="nm-inline-actions">
              {requiresAuth ? (
                <Link href="/login" className="nm-link-btn nm-link-btn-primary">Se connecter</Link>
              ) : (
                <Link href="/onboarding" className="nm-link-btn nm-link-btn-primary">Compléter le profil</Link>
              )}
            </div>
          </div>
        )}

        {!loading && !error && !selectedChoice && meals.length === 0 && (
          <div className="nm-card">
            <p className="nm-sub">Aucune recette sûre n&apos;est disponible pour ce profil actuellement.</p>
          </div>
        )}

        {!loading && !error && !selectedChoice && meals.length > 0 && (
          <RecommendationList
            meals={meals}
            aiExplanationApplied={response?.aiExplanationApplied === true}
            choosingMealId={choosingMealId}
            aiMessage={explanationError || (response && !response.aiExplanationApplied ? labelAIReason(aiReason) : "")}
            refreshingExplanation={refreshingExplanation}
            onChoose={(mealId) => void handleChoose(mealId)}
            onRetryExplanation={canRetryExplanation ? () => void handleRetryExplanation() : undefined}
          />
        )}
      </section>
    </main>
  );
}
