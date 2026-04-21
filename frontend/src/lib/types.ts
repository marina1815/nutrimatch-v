export type Sex = "male" | "female";

export type Goal =
  | "weight_loss"
  | "muscle_gain"
  | "weight_maintenance"
  | "medical_diet"
  | "energy_maintenance";

export type ActivityLevel =
  | "sedentary"
  | "light"
  | "moderate"
  | "active";

export type LifestyleType =
  | "student"
  | "employee"
  | "athlete"
  | "mixed"
  | "other";

export type Intolerance =
  | "dairy"
  | "egg"
  | "gluten"
  | "grain"
  | "peanut"
  | "seafood"
  | "sesame"
  | "shellfish"
  | "soy"
  | "sulfite"
  | "tree_nut"
  | "wheat";

export type Condition =
  | "diabetes"
  | "hypertension"
  | "cardiac"
  | "renal_failure"
  | "hypercholesterolemia"
  | "digestive_sensitivity"
  | "other";

export type MealStyle =
  | "traditional"
  | "healthy"
  | "middle eastern"
  | "modern"
  | "cold"
  | "quick"
  | "balanced"
  | "high-protein"
  | "low-sodium"
  | "low-sugar";

export type MealType =
  | "main_course"
  | "side_dish"
  | "breakfast"
  | "lunch"
  | "dinner"
  | "snack"
  | "salad"
  | "soup"
  | "dessert"
  | "appetizer"
  | "beverage";

export type Cuisine =
  | "african"
  | "american"
  | "asian"
  | "mediterranean"
  | "middle_eastern"
  | "european"
  | "mexican";

export type ChronicDisease =
  | "diabetes"
  | "hypertension"
  | "cardiac"
  | "renal_failure"
  | "other";

export interface PersonalInfo {
  fullName: string;
  age: number | "";
  sex: Sex | "";
  weight: number | "";
  height: number | "";
  profession: string;
  city: string;
}

export interface LifestyleInfo {
  activityLevel: ActivityLevel | "";
  lifestyleType: LifestyleType | "";
  goal: Goal | "";
  maxReadyTime: number | "";
}

export interface PreferencesInfo {
  likes: string[];
  dislikes: string[];
  mealStyles: MealStyle[];
  mealTypes: MealType[];
  preferredCuisines: Cuisine[];
  excludedCuisines: Cuisine[];
  mealsPerDay: number | "";
}

export interface ConstraintsInfo {
  allergies: Intolerance[];
  conditions: Condition[];
  excludedIngredients: string[];
  hasChronicDisease: boolean;
  chronicDiseases: ChronicDisease[];
  takesMedication: boolean;
  medications: string;
}

export interface UserProfile {
  personal: PersonalInfo;
  lifestyle: LifestyleInfo;
  preferences: PreferencesInfo;
  constraints: ConstraintsInfo;
}

export interface MealRecommendation {
  id: string;
  title: string;
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
  sugar?: number;
  sodiumMg?: number;
  tags: string[];
  description: string;
  ingredients: string[];
  matchReason: string;
  source?: string;
  score?: number;
}

export interface HealthMetrics {
  bmi: number;
  bmiCategory: string;
  bmr: number;
  estimatedCalories: number;
}

export interface UserProfileResponse extends UserProfile {
  profileId: string;
}

export interface NutritionProfile {
  profileId: string;
  bmi: number;
  bmiCategory: string;
  bmr: number;
  estimatedCalories: number;
  targetCalories: number;
  targetProteinGrams: number;
  targetCarbsGrams: number;
  targetFatGrams: number;
  maxMealCalories: number;
  minProteinPerMeal: number;
  maxCarbsPerMeal: number;
  maxFatPerMeal: number;
  maxSugarPerMeal: number;
  maxSodiumMgPerMeal: number;
  derivedRestrictions: string[];
  derivedExcluded: string[];
  recommendedMealStyles: string[];
  metadata: Record<string, unknown>;
}

export interface RecommendationExplanation {
  runId: string;
  profileId: string;
  mealId: string;
  explanation: string;
  acceptedReasons: string[];
  rejectedReasons: string[];
  scoreBreakdown: Record<string, unknown>;
  filterDecisions: Record<string, unknown>;
  sourceProvenance: Record<string, unknown>;
}

export interface RecommendationTraceMeal {
  mealId: string;
  title: string;
  accepted: boolean;
  finalRank: number;
  finalScore: number;
  acceptedReasons: string[];
  rejectedReasons: string[];
  scoreBreakdown: Record<string, unknown>;
  filterDecisions: Record<string, unknown>;
  sourceProvenance: Record<string, unknown>;
}

export interface RecommendationTrace {
  runId: string;
  profileId: string;
  status: string;
  sourceSummary: Record<string, unknown>;
  decisionSummary: Record<string, unknown>;
  externalTrace: Record<string, unknown>;
  candidates: RecommendationTraceMeal[];
}
