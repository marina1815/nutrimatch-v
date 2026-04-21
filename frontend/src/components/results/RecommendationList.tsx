import { MealRecommendation, RecommendationExplanation } from "@/lib/types";
import { MealCard } from "./MealCard";

type Props = {
  meals: MealRecommendation[];
  explanationsByMealId?: Record<string, RecommendationExplanation>;
  loadingExplanationMealId?: string | null;
  onExplain?: (mealId: string) => void;
};

export function RecommendationList({
  meals,
  explanationsByMealId = {},
  loadingExplanationMealId = null,
  onExplain,
}: Props) {
  return (
    <div className="nm-results-grid">
      {meals.map((meal) => (
        <MealCard
          key={meal.id}
          meal={meal}
          explanation={explanationsByMealId[meal.id]}
          loadingExplanation={loadingExplanationMealId === meal.id}
          onExplain={onExplain ? () => onExplain(meal.id) : undefined}
        />
      ))}
    </div>
  );
}
