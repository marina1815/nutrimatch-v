import { MealRecommendation, RecommendationExplanation } from "@/lib/types";

type Props = {
  meal: MealRecommendation;
  explanation?: RecommendationExplanation;
  loadingExplanation?: boolean;
  onExplain?: () => void;
};

export function MealCard({ meal, explanation, loadingExplanation, onExplain }: Props) {
  return (
    <div className="nm-card">
      <div className="nm-meal-top">
        <h3>{meal.title}</h3>
        <span className="nm-badge">{meal.calories} kcal</span>
      </div>

      <p className="nm-muted">{meal.description}</p>

      <div className="nm-macros">
        <span>Protein: {meal.protein}g</span>
        <span>Carbs: {meal.carbs}g</span>
        <span>Fat: {meal.fat}g</span>
      </div>

      <div className="nm-tags">
        {meal.tags.map((tag) => (
          <span key={tag} className="nm-tag">{tag}</span>
        ))}
      </div>

      <p className="nm-reason">{meal.matchReason}</p>

      {onExplain && (
        <div className="nm-inline-actions">
          <button
            type="button"
            className="nm-link-btn"
            onClick={onExplain}
            disabled={loadingExplanation}
          >
            {loadingExplanation ? "Loading explanation..." : "View explanation"}
          </button>
        </div>
      )}

      {explanation && (
        <div className="nm-explain-box">
          <p className="nm-reason">{explanation.explanation}</p>
          <p className="nm-muted">
            Accepted: {explanation.acceptedReasons.join(", ") || "-"}
          </p>
          <p className="nm-muted">
            Rejected: {explanation.rejectedReasons.join(", ") || "-"}
          </p>
        </div>
      )}
    </div>
  );
}
