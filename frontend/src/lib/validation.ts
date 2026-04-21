import { sanitizeProfile } from "@/lib/profile-normalization";
import { UserProfile } from "@/lib/types";

const MAX_FLEXIBLE_SIGNAL_COUNT = 40;
const MAX_PROFILE_TEXT_BUDGET = 1200;

export type ProfileErrors = {
  personal?: Partial<
    Record<"fullName" | "age" | "sex" | "weight" | "height" | "profession" | "city", string>
  >;
  lifestyle?: Partial<Record<"activityLevel" | "lifestyleType" | "goal" | "maxReadyTime", string>>;
  preferences?: Partial<Record<"likes" | "dislikes" | "mealStyles" | "mealTypes" | "preferredCuisines" | "excludedCuisines" | "mealsPerDay", string>>;
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
      personal.age = "Age invalide";
    }
    if (!normalized.personal.sex) personal.sex = "Selectionne le sexe";
    if (!normalized.personal.weight || Number(normalized.personal.weight) < 20 || Number(normalized.personal.weight) > 400) {
      personal.weight = "Poids invalide";
    }
    if (!normalized.personal.height || Number(normalized.personal.height) < 80 || Number(normalized.personal.height) > 250) {
      personal.height = "Taille invalide";
    }
    if (!normalized.personal.profession) personal.profession = "La profession est requise";
    if (!normalized.personal.city) personal.city = "La ville est requise";

    if (Object.keys(personal).length > 0) errors.personal = personal;
  }

  if (step === 1) {
    const lifestyle: NonNullable<ProfileErrors["lifestyle"]> = {};

    if (!normalized.lifestyle.activityLevel) lifestyle.activityLevel = "Choisis un niveau d'activite";
    if (!normalized.lifestyle.lifestyleType) lifestyle.lifestyleType = "Choisis un mode de vie";
    if (!normalized.lifestyle.goal) lifestyle.goal = "Choisis un objectif";
    if (!normalized.lifestyle.maxReadyTime || Number(normalized.lifestyle.maxReadyTime) < 5 || Number(normalized.lifestyle.maxReadyTime) > 240) {
      lifestyle.maxReadyTime = "Le temps maximal doit etre entre 5 et 240 minutes";
    }

    if (Object.keys(lifestyle).length > 0) errors.lifestyle = lifestyle;
  }

  if (step === 2) {
    const preferences: NonNullable<ProfileErrors["preferences"]> = {};
    const likeOverlap = normalized.preferences.likes.some((item) =>
      normalized.preferences.dislikes.some((other) => other.toLowerCase() === item.toLowerCase()),
    );
    const cuisineOverlap = normalized.preferences.preferredCuisines.some((item) =>
      normalized.preferences.excludedCuisines.includes(item),
    );

    if (!normalized.preferences.mealsPerDay || Number(normalized.preferences.mealsPerDay) < 1 || Number(normalized.preferences.mealsPerDay) > 8) {
      preferences.mealsPerDay = "Indique le nombre de repas par jour";
    }
    if (likeOverlap) {
      preferences.dislikes = "Un ingredient ne peut pas etre aime et non aime en meme temps";
    }
    if (cuisineOverlap) {
      preferences.excludedCuisines = "Une cuisine ne peut pas etre a la fois preferee et exclue";
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
      normalized.personal.profession.length +
      normalized.personal.city.length +
      normalized.constraints.medications.trim().length +
      normalized.preferences.likes.reduce((sum, item) => sum + item.length, 0) +
      normalized.preferences.dislikes.reduce((sum, item) => sum + item.length, 0) +
      normalized.constraints.excludedIngredients.reduce((sum, item) => sum + item.length, 0);

    if (normalized.constraints.hasChronicDisease && normalized.constraints.chronicDiseases.length === 0) {
      constraints.chronicDiseases = "Selectionne au moins une maladie chronique";
    }
    if (!normalized.constraints.hasChronicDisease && normalized.constraints.chronicDiseases.length > 0) {
      constraints.chronicDiseases = "Retire les maladies chroniques ou active l'option correspondante";
    }
    if (normalized.constraints.takesMedication && !normalized.constraints.medications.trim()) {
      constraints.medications = "Precise les medicaments";
    }
    if (!normalized.constraints.takesMedication && normalized.constraints.medications.trim()) {
      constraints.medications = "Retire les medicaments ou active l'option correspondante";
    }
    if (exclusionOverlap) {
      constraints.excludedIngredients = "Un ingredient aime ne peut pas etre exclu";
    }
    if (flexibleSignalCount > MAX_FLEXIBLE_SIGNAL_COUNT) {
      constraints.excludedIngredients = "Reduis le nombre total d'ingredients libres et de preferences saisies";
    }
    if (textBudget > MAX_PROFILE_TEXT_BUDGET) {
      constraints.excludedIngredients = "Le profil est trop verbeux pour une recherche fiable, simplifie les saisies libres";
    }

    if (Object.keys(constraints).length > 0) errors.constraints = constraints;
  }

  return errors;
}
