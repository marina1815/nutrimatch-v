import {
  ActivityLevel,
  ChronicDisease,
  Condition,
  Cuisine,
  Goal,
  Intolerance,
  LifestyleType,
  MealStyle,
  MealType,
  Sex,
} from "@/lib/types";

type Option<T extends string = string> = {
  value: T;
  label: string;
};

export const ACTIVITY_OPTIONS: Option<ActivityLevel>[] = [
  { value: "sedentary", label: "Sedentaire" },
  { value: "light", label: "Legerement actif" },
  { value: "moderate", label: "Modere actif" },
  { value: "active", label: "Tres actif" },
];

export const LIFESTYLE_OPTIONS: Option<LifestyleType>[] = [
  { value: "student", label: "Etudiant" },
  { value: "employee", label: "Employe" },
  { value: "athlete", label: "Sportif" },
  { value: "mixed", label: "Mixte" },
  { value: "other", label: "Autre" },
];

export const GOAL_OPTIONS: Option<Goal>[] = [
  { value: "weight_loss", label: "Perte de poids" },
  { value: "muscle_gain", label: "Gain de muscle" },
  { value: "weight_maintenance", label: "Maintien du poids" },
  { value: "medical_diet", label: "Regime medical" },
  { value: "energy_maintenance", label: "Maintien d energie" },
];

export const SEX_OPTIONS: Option<Sex>[] = [
  { value: "male", label: "Homme" },
  { value: "female", label: "Femme" },
];

export const MEAL_STYLE_OPTIONS: Option<MealStyle>[] = [
  { value: "traditional", label: "Traditionnel" },
  { value: "healthy", label: "Recettes saines" },
  { value: "middle eastern", label: "Oriental" },
  { value: "modern", label: "Moderne" },
  { value: "cold", label: "Repas froids" },
  { value: "quick", label: "Rapide" },
  { value: "balanced", label: "Equilibre" },
  { value: "high-protein", label: "Riche en proteines" },
  { value: "low-sodium", label: "Faible en sodium" },
  { value: "low-sugar", label: "Faible en sucre" },
];

export const MEAL_TYPE_OPTIONS: Option<MealType>[] = [
  { value: "main_course", label: "Plat principal" },
  { value: "side_dish", label: "Accompagnement" },
  { value: "breakfast", label: "Petit dejeuner" },
  { value: "lunch", label: "Dejeuner" },
  { value: "dinner", label: "Diner" },
  { value: "snack", label: "Collation" },
  { value: "salad", label: "Salade" },
  { value: "soup", label: "Soupe" },
  { value: "dessert", label: "Dessert" },
  { value: "appetizer", label: "Entree" },
  { value: "beverage", label: "Boisson" },
];

export const CUISINE_OPTIONS: Option<Cuisine>[] = [
  { value: "african", label: "Africaine" },
  { value: "american", label: "Americaine" },
  { value: "asian", label: "Asiatique" },
  { value: "mediterranean", label: "Mediterraneenne" },
  { value: "middle_eastern", label: "Moyen-Orient" },
  { value: "european", label: "Europeenne" },
  { value: "mexican", label: "Mexicaine" },
];

export const COMMON_ALLERGIES: Option<Intolerance>[] = [
  { value: "peanut", label: "Arachides" },
  { value: "dairy", label: "Lait" },
  { value: "egg", label: "Oeufs" },
  { value: "soy", label: "Soja" },
  { value: "seafood", label: "Poisson" },
  { value: "shellfish", label: "Fruits de mer" },
  { value: "gluten", label: "Gluten" },
  { value: "sesame", label: "Sesame" },
  { value: "tree_nut", label: "Fruits a coque" },
  { value: "wheat", label: "Ble" },
];

export const COMMON_CONDITIONS: Option<Condition>[] = [
  { value: "diabetes", label: "Diabete" },
  { value: "hypertension", label: "Hypertension" },
  { value: "cardiac", label: "Maladie cardiaque" },
  { value: "renal_failure", label: "Insuffisance renale" },
  { value: "hypercholesterolemia", label: "Cholesterol eleve" },
  { value: "digestive_sensitivity", label: "Sensibilite digestive" },
  { value: "other", label: "Autre" },
];

export const CHRONIC_DISEASE_OPTIONS: Option<ChronicDisease>[] = [
  { value: "diabetes", label: "Diabete" },
  { value: "hypertension", label: "Hypertension" },
  { value: "cardiac", label: "Maladie cardiaque" },
  { value: "renal_failure", label: "Insuffisance renale" },
  { value: "other", label: "Autre" },
];
