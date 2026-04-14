export const ACTIVITY_OPTIONS = [
  { value: "sedentary", label: "Sédentaire" },
  { value: "light", label: "Légèrement actif" },
  { value: "moderate", label: "Modérément actif" },
  { value: "active", label: "Très actif" },
];

export const LIFESTYLE_OPTIONS = [
  { value: "student", label: "Étudiant" },
  { value: "employee", label: "Employé" },
  { value: "athlete", label: "Sportif" },
  { value: "mixed", label: "Mixte" },
  { value: "other", label: "Autre" },
];

export const GOAL_OPTIONS = [
  { value: "weight_loss", label: "Perte de poids" },
  { value: "muscle_gain", label: "Gain de muscle" },
  { value: "weight_maintenance", label: "Maintien de poids" },
  { value: "medical_diet", label: "Régime médical" },
  { value: "energy_maintenance", label: "Maintien d’énergie" },
];

export const SEX_OPTIONS = [
  { value: "male", label: "Homme" },
  { value: "female", label: "Femme" },
];

export const DIET_OPTIONS = [
  { value: "gluten free", label: "Sans Gluten" },
  { value: "ketogenic", label: "Cétogène (Keto)" },
  { value: "vegetarian", label: "Végétarien" },
  { value: "lacto-vegetarian", label: "Lacto-Végétarien" },
  { value: "ovo-vegetarian", label: "Ovo-Végétarien" },
  { value: "vegan", label: "Végétalien" },
  { value: "pescetarian", label: "Pescétarien" },
  { value: "paleo", label: "Paléo" },
  { value: "primal", label: "Primal" },
  { value: "whole30", label: "Whole30" },
];

export const CUISINE_OPTIONS = [
  { value: "african", label: "Africaine" },
  { value: "american", label: "Américaine" },
  { value: "british", label: "Britannique" },
  { value: "cajun", label: "Cajun" },
  { value: "caribbean", label: "Caribéenne" },
  { value: "chinese", label: "Chinoise" },
  { value: "eastern european", label: "Europe de l'Est" },
  { value: "european", label: "Européenne" },
  { value: "french", label: "Française" },
  { value: "german", label: "Allemande" },
  { value: "greek", label: "Grecque" },
  { value: "indian", label: "Indienne" },
  { value: "irish", label: "Irlandaise" },
  { value: "italian", label: "Italienne" },
  { value: "japanese", label: "Japonaise" },
  { value: "jewish", label: "Juive" },
  { value: "korean", label: "Coréenne" },
  { value: "latin american", label: "Latino-américaine" },
  { value: "mediterranean", label: "Méditerranéenne" },
  { value: "mexican", label: "Mexicaine" },
  { value: "middle eastern", label: "Moyen-Orient" },
  { value: "nordic", label: "Nordique" },
  { value: "southern", label: "Sud-américaine" },
  { value: "spanish", label: "Espagnole" },
  { value: "thai", label: "Thaïlandaise" },
  { value: "vietnamese", label: "Vietnamienne" },
];

export const INTOLERANCE_OPTIONS = [
  { value: "dairy", label: "Produits laitiers" },
  { value: "egg", label: "Œufs" },
  { value: "gluten", label: "Gluten" },
  { value: "grain", label: "Grains" },
  { value: "peanut", label: "Arachides" },
  { value: "seafood", label: "Poisson" },
  { value: "sesame", label: "Sésame" },
  { value: "shellfish", label: "Fruits de mer" },
  { value: "soy", label: "Soja" },
  { value: "sulfite", label: "Sulfite" },
  { value: "tree nut", label: "Fruits à coque" },
  { value: "wheat", label: "Blé" },
];

export const COMMON_ALLERGIES = INTOLERANCE_OPTIONS.map((o) => o.label);

export const COMMON_CONDITIONS = [
  "Diabète",
  "Hypertension",
  "Cholestérol élevé",
  "Sensibilité digestive",
];

export const CHRONIC_DISEASE_OPTIONS = [
  { value: "diabetes", label: "Diabète" },
  { value: "hypertension", label: "Hypertension" },
  { value: "cardiac", label: "Maladie cardiaque" },
  { value: "renal_failure", label: "Insuffisance rénale" },
  { value: "other", label: "Autre" },
];

export const MEAL_STYLE_OPTIONS = CUISINE_OPTIONS.map((o) => o.label);