import { sanitizeProfile } from "@/lib/profile-normalization";
import { UserProfile } from "@/lib/types";

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

    if (normalized.constraints.hasChronicDisease && normalized.constraints.chronicDiseases.length === 0) {
      constraints.chronicDiseases = "Selectionne au moins une maladie chronique";
    }
    if (normalized.constraints.takesMedication && !normalized.constraints.medications.trim()) {
      constraints.medications = "Precise les medicaments";
    }
    if (exclusionOverlap) {
      constraints.excludedIngredients = "Un ingredient aime ne peut pas etre exclu";
    }

    if (Object.keys(constraints).length > 0) errors.constraints = constraints;
  }

  return errors;
}
