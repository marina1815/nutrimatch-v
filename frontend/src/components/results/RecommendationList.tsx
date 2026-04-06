import { MealRecommendation } from "@/lib/types";
import { MealCard } from "./MealCard";

export function RecommendationList({ meals }: { meals: MealRecommendation[] }) {
  return (
    <div className="nm-results-grid">
      {meals.map((meal) => (
        <MealCard key={meal.id} meal={meal} />
      ))}
    </div>
  );
}