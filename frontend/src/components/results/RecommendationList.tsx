import { MealRecommendation } from "@/lib/types";
import { MealCard } from "./MealCard";

type Props = {
  meals: MealRecommendation[];
  aiExplanationApplied?: boolean;
  choosingMealId?: string | null;
  aiMessage?: string;
  refreshingExplanation?: boolean;
  onChoose?: (mealId: string) => void;
  onRetryExplanation?: () => void;
};

export function RecommendationList({
  meals,
  aiExplanationApplied = false,
  choosingMealId = null,
  aiMessage = "",
  refreshingExplanation = false,
  onChoose,
  onRetryExplanation,
}: Props) {
  return (
    <div className="nm-stack">
      {!aiExplanationApplied && aiMessage && (
        <div className="nm-card">
          <strong>Explication IA indisponible</strong>
          <p className="nm-muted">{aiMessage}</p>
          {onRetryExplanation && (
            <button
              type="button"
              className="nm-link-btn"
              onClick={onRetryExplanation}
              disabled={refreshingExplanation}
            >
              {refreshingExplanation ? "Nouvel essai IA..." : "Reessayer les explications IA"}
            </button>
          )}
        </div>
      )}
      <div className="nm-results-grid">
        {meals.map((meal) => (
          <MealCard
            key={meal.id}
            meal={meal}
            aiExplanationApplied={aiExplanationApplied}
            choosing={choosingMealId === meal.id}
            onChoose={onChoose ? () => onChoose(meal.id) : undefined}
          />
        ))}
      </div>
    </div>
  );
}
