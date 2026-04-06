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
}

export interface PreferencesInfo {
  likes: string[];
  dislikes: string[];
  mealStyles: string[];
  mealsPerDay: number | "";
}

export interface ConstraintsInfo {
  allergies: string[];
  conditions: string[];
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
  tags: string[];
  description: string;
  ingredients: string[];
  matchReason: string;
}

export interface HealthMetrics {
  bmi: number;
  bmiCategory: string;
  bmr: number;
  estimatedCalories: number;
}