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

export type Intolerance = string;

export type Condition =
  | "diabetes"
  | "hypertension"
  | "cardiac"
  | "renal_failure"
  | "hypercholesterolemia"
  | "digestive_sensitivity";

export type ChronicDisease =
  | "diabetes"
  | "hypertension"
  | "cardiac"
  | "renal_failure"
  | "hypercholesterolemia"
  | "digestive_sensitivity";

export interface PersonalInfo {
  fullName: string;
  age: number | "";
  sex: Sex | "";
  weight: number | "";
  height: number | "";
}

export interface LifestyleInfo {
  activityLevel: ActivityLevel | "";
  lifestyleType: LifestyleType | "";
  goal: Goal | "";
}

export interface PreferencesInfo {
  likes: string[];
  dislikes: string[];
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

export interface ConstraintsResponse extends ConstraintsInfo {
  medicationsRedacted?: boolean;
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
  ingredients: CatalogOption[];
  matchReason: string;
  source?: string;
  score?: number;
  nutritionConfidence?: "estimated" | "reported" | string;
  nutritionSource?: string;
  safetyWarnings?: string[];
  aiExplanation?: string;
}

export interface RecommendationResponse {
  runId: string;
  profileId: string;
  meals: MealRecommendation[];
  activeChoice?: MealChoiceResponse;
  generatedAt: string;
  validUntil: string;
  nextRefreshAt: string;
  secondsUntilRefresh: number;
  selectionMode: string;
  aiExplanationApplied: boolean;
  aiValidationApplied: boolean;
  aiRejectedMealCount: number;
  aiReplacementCount: number;
  aiSkippedReason?: string;
  aiOutputIgnoredReason?: string;
}

export interface MealSubstitution {
  from: string;
  to: string;
  reason: string;
}

export interface MealChoiceResponse {
  profileId: string;
  meal: MealRecommendation;
  preparationGuide: string;
  substitutions: MealSubstitution[];
  aiApplied: boolean;
  aiSkippedReason?: string;
  aiOutputIgnoredReason?: string;
  chosenAt: string;
  excludedUntil: string;
}

export interface HealthMetrics {
  bmi: number;
  bmiCategory: string;
  bmr: number;
  estimatedCalories: number;
}

export interface UserProfileResponse extends Omit<UserProfile, "constraints"> {
  profileId: string;
  constraints: ConstraintsResponse;
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
  metadata: Record<string, unknown>;
}

export interface CurrentSession {
  userId: string;
  email: string;
  fullName: string;
  sessionId: string;
  authMethod: string;
  hasProfile: boolean;
  profileId: string;
}

export interface AuthSession {
  id: string;
  authMethod: string;
  expiresAt: string;
  idleExpiresAt: string;
  createdAt: string;
  lastSeenAt: string;
  revoked: boolean;
  current: boolean;
}

export interface CatalogOption {
  value: string;
  label: string;
  source?: string;
}

export interface ProfileTaxonomy {
  allergies: CatalogOption[];
  conditions: CatalogOption[];
  chronicDiseases: CatalogOption[];
}

export interface RecommendationExplanation {
  runId: string;
  profileId: string;
  mealId: string;
  explanation: string;
  aiExplanation?: string;
  aiExplanationApplied?: boolean;
  aiSkippedReason?: string;
  aiOutputIgnoredReason?: string;
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

