import { sanitizeDisplayText } from "@/lib/text-sanitization";
import { MealRecommendation } from "@/lib/types";

type Props = {
  meal: MealRecommendation;
  aiExplanationApplied?: boolean;
  choosing?: boolean;
  onChoose?: () => void;
};

export function MealCard({
  meal,
  aiExplanationApplied = false,
  choosing = false,
  onChoose,
}: Props) {
  const ingredients = meal.ingredients
    .map((ingredient) => sanitizeDisplayText(ingredient.label || ingredient.value))
    .filter(Boolean);
  const explanation = aiExplanationApplied ? sanitizeDisplayText(meal.aiExplanation || "") : "";

  return (
    <div className="nm-card">
      <div className="nm-meal-top">
        <h3>{sanitizeDisplayText(meal.title)}</h3>
      </div>

      {ingredients.length > 0 && (
        <div className="nm-ingredients">
          <strong>Ingr&eacute;dients</strong>
          <div className="nm-ingredient-list">
            {ingredients.map((ingredient) => (
              <span key={ingredient} className="nm-ingredient">{ingredient}</span>
            ))}
          </div>
        </div>
      )}

      <div className="nm-explain-box">
        <strong>Explication IA</strong>
        {explanation ? (
          <p className="nm-reason">{explanation}</p>
        ) : (
          <p className="nm-muted">Aucune explication IA disponible pour cette recette.</p>
        )}
      </div>

      {meal.nutritionConfidence === "estimated" && (
        <p className="nm-muted">Nutrition estim&eacute;e depuis le catalogue local.</p>
      )}

      {onChoose && (
        <div className="nm-inline-actions">
          <button
            type="button"
            className="nm-link-btn nm-link-btn-primary"
            onClick={onChoose}
            disabled={choosing}
          >
            {choosing ? "Choix en cours..." : "Choisir cette recette"}
          </button>
        </div>
      )}
    </div>
  );
}
