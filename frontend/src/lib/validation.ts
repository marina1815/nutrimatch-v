import { sanitizeProfile } from "@/lib/profile-normalization";
import { UserProfile } from "@/lib/types";

const MAX_FLEXIBLE_SIGNAL_COUNT = 40;
const MAX_PROFILE_TEXT_BUDGET = 1200;

export type ProfileErrors = {
  personal?: Partial<
    Record<"fullName" | "age" | "sex" | "weight" | "height", string>
  >;
  lifestyle?: Partial<Record<"activityLevel" | "lifestyleType" | "goal", string>>;
  preferences?: Partial<Record<"likes" | "dislikes", string>>;
  constraints?: Partial<
    Record<"allergies" | "conditions" | "excludedIngredients" | "chronicDiseases" | "medications", string>
  >;
};

export function validateStep(step: number, data: UserProfile): ProfileErrors {
  const normalized = sanitizeProfile(data);
  const errors: ProfileErrors = {};

  if (step === 0) {
    const personal: NonNullable<ProfileErrors["personal"]> = {};

    if (normalized.personal.fullName.length < 2) personal.fullName = "Le nom complet est requis";
    if (!normalized.personal.age || Number(normalized.personal.age) < 10 || Number(normalized.personal.age) > 120) {
      personal.age = "Âge invalide";
    }
    if (!normalized.personal.sex) personal.sex = "Sélectionne le sexe";
    if (!normalized.personal.weight || Number(normalized.personal.weight) < 20 || Number(normalized.personal.weight) > 400) {
      personal.weight = "Poids invalide";
    }
    if (!normalized.personal.height || Number(normalized.personal.height) < 80 || Number(normalized.personal.height) > 250) {
      personal.height = "Taille invalide";
    }
    if (Object.keys(personal).length > 0) errors.personal = personal;
  }

  if (step === 1) {
    const lifestyle: NonNullable<ProfileErrors["lifestyle"]> = {};

    if (!normalized.lifestyle.activityLevel) lifestyle.activityLevel = "Choisis un niveau d'activité";
    if (!normalized.lifestyle.lifestyleType) lifestyle.lifestyleType = "Choisis un mode de vie";
    if (!normalized.lifestyle.goal) lifestyle.goal = "Choisis un objectif";

    if (Object.keys(lifestyle).length > 0) errors.lifestyle = lifestyle;
  }

  if (step === 2) {
    const preferences: NonNullable<ProfileErrors["preferences"]> = {};
    const likeOverlap = normalized.preferences.likes.some((item) =>
      normalized.preferences.dislikes.some((other) => other.toLowerCase() === item.toLowerCase()),
    );

    if (likeOverlap) {
      preferences.dislikes = "Un ingrédient ne peut pas être aimé et non aimé en même temps";
    }

    if (Object.keys(preferences).length > 0) errors.preferences = preferences;
  }

  if (step === 3) {
    const constraints: NonNullable<ProfileErrors["constraints"]> = {};
    const exclusionOverlap = normalized.constraints.excludedIngredients.some((item) =>
      normalized.preferences.likes.some((liked) => liked.toLowerCase() === item.toLowerCase()),
    );
    const flexibleSignalCount =
      normalized.preferences.likes.length +
      normalized.preferences.dislikes.length +
      normalized.constraints.excludedIngredients.length;
    const textBudget =
      normalized.personal.fullName.length +
      normalized.constraints.medications.trim().length +
      normalized.preferences.likes.reduce((sum, item) => sum + item.length, 0) +
      normalized.preferences.dislikes.reduce((sum, item) => sum + item.length, 0) +
      normalized.constraints.excludedIngredients.reduce((sum, item) => sum + item.length, 0);

    if (normalized.constraints.hasChronicDisease && normalized.constraints.chronicDiseases.length === 0) {
      constraints.chronicDiseases = "Sélectionne au moins une maladie chronique";
    }
    if (!normalized.constraints.hasChronicDisease && normalized.constraints.chronicDiseases.length > 0) {
      constraints.chronicDiseases = "Retire les maladies chroniques ou active l'option correspondante";
    }
    if (normalized.constraints.takesMedication && !normalized.constraints.medications.trim()) {
      constraints.medications = "Précise les médicaments";
    }
    if (!normalized.constraints.takesMedication && normalized.constraints.medications.trim()) {
      constraints.medications = "Retire les médicaments ou active l'option correspondante";
    }
    if (exclusionOverlap) {
      constraints.excludedIngredients = "Un ingrédient aimé ne peut pas être exclu";
    }
    if (flexibleSignalCount > MAX_FLEXIBLE_SIGNAL_COUNT) {
      constraints.excludedIngredients = "Réduis le nombre total d'ingrédients et préférences";
    }
    if (textBudget > MAX_PROFILE_TEXT_BUDGET) {
      constraints.excludedIngredients = "Le profil est trop verbeux pour une recherche fiable";
    }

    if (Object.keys(constraints).length > 0) errors.constraints = constraints;
  }

  return errors;
}
