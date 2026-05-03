import {
  ActivityLevel,
  ChronicDisease,
  Condition,
  Goal,
  Intolerance,
  LifestyleType,
  Sex,
} from "@/lib/types";

export type Option<T extends string = string> = {
  value: T;
  label: string;
};

export const ACTIVITY_OPTIONS: Option<ActivityLevel>[] = [
  { value: "sedentary", label: "Sedentaire" },
  { value: "light", label: "Legerement actif" },
  { value: "moderate", label: "Moderement actif" },
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
  { value: "energy_maintenance", label: "Maintien d'energie" },
];

export const SEX_OPTIONS: Option<Sex>[] = [
  { value: "male", label: "Homme" },
  { value: "female", label: "Femme" },
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
  { value: "hypercholesterolemia", label: "Hypercholesterolemie" },
  { value: "digestive_sensitivity", label: "Sensibilite digestive" },
];

export const CHRONIC_DISEASE_OPTIONS: Option<ChronicDisease>[] = [
  { value: "diabetes", label: "Diabete" },
  { value: "hypertension", label: "Hypertension" },
  { value: "cardiac", label: "Maladie cardiaque" },
  { value: "renal_failure", label: "Insuffisance renale" },
  { value: "hypercholesterolemia", label: "Hypercholesterolemie" },
  { value: "digestive_sensitivity", label: "Sensibilite digestive" },
];
