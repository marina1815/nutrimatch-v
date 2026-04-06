import { MealRecommendation } from "@/lib/types";

export function MealCard({ meal }: { meal: MealRecommendation }) {
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
    </div>
  );
}