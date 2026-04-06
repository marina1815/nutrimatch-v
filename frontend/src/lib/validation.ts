import { UserProfile } from "@/lib/types";

export type ProfileErrors = {
  personal?: Partial<
    Record<"fullName" | "age" | "sex" | "weight" | "height" | "profession" | "city", string>
  >;
  lifestyle?: Partial<Record<"activityLevel" | "lifestyleType" | "goal", string>>;
  preferences?: Partial<Record<"mealsPerDay", string>>;
  constraints?: Partial<Record<"chronicDiseases" | "medications", string>>;
};

export function validateStep(step: number, data: UserProfile): ProfileErrors {
  const errors: ProfileErrors = {};

  if (step === 0) {
    const personal: NonNullable<ProfileErrors["personal"]> = {};

    if (!data.personal.fullName.trim()) personal.fullName = "Le nom est requis";
    if (!data.personal.age || Number(data.personal.age) < 10) personal.age = "Âge invalide";
    if (!data.personal.sex) personal.sex = "Sélectionnez le sexe";
    if (!data.personal.weight || Number(data.personal.weight) < 20) personal.weight = "Poids invalide";
    if (!data.personal.height || Number(data.personal.height) < 80) personal.height = "Taille invalide";
    if (!data.personal.profession.trim()) personal.profession = "La profession est requise";
    if (!data.personal.city.trim()) personal.city = "La ville est requise";

    if (Object.keys(personal).length > 0) errors.personal = personal;
  }

  if (step === 1) {
    const lifestyle: NonNullable<ProfileErrors["lifestyle"]> = {};

    if (!data.lifestyle.activityLevel) lifestyle.activityLevel = "Choisissez un niveau d’activité";
    if (!data.lifestyle.lifestyleType) lifestyle.lifestyleType = "Choisissez un mode de vie";
    if (!data.lifestyle.goal) lifestyle.goal = "Choisissez un objectif";

    if (Object.keys(lifestyle).length > 0) errors.lifestyle = lifestyle;
  }

  if (step === 2) {
    const preferences: NonNullable<ProfileErrors["preferences"]> = {};

    if (!data.preferences.mealsPerDay || Number(data.preferences.mealsPerDay) < 1) {
      preferences.mealsPerDay = "Indiquez le nombre de repas par jour";
    }

    if (Object.keys(preferences).length > 0) errors.preferences = preferences;
  }

  if (step === 3) {
    const constraints: NonNullable<ProfileErrors["constraints"]> = {};

    if (data.constraints.hasChronicDisease && data.constraints.chronicDiseases.length === 0) {
      constraints.chronicDiseases = "Sélectionnez au moins une maladie chronique";
    }

    if (data.constraints.takesMedication && !data.constraints.medications.trim()) {
      constraints.medications = "Précisez les médicaments";
    }

    if (Object.keys(constraints).length > 0) errors.constraints = constraints;
  }

  return errors;
}