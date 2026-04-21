import { UserProfile } from "@/lib/types";

const LIMITS = {
  fullName: 120,
  profession: 120,
  city: 120,
  medications: 250,
  itemLength: 50,
  likes: 25,
  dislikes: 25,
  mealStyles: 20,
  mealTypes: 6,
  cuisines: 8,
  allergies: 20,
  conditions: 20,
  excludedIngredients: 30,
  chronicDiseases: 10,
} as const;

function normalizeText(value: string, maxLength: number): string {
  return value.replace(/\s+/g, " ").trim().slice(0, maxLength);
}

function uniqueList(values: string[], maxItems: number, maxLength: number): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const value of values) {
    const cleaned = normalizeText(value, maxLength);
    if (!cleaned) {
      continue;
    }

    const key = cleaned.toLowerCase();
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    out.push(cleaned);

    if (out.length >= maxItems) {
      break;
    }
  }

  return out;
}

export function parseCommaSeparatedList(raw: string, maxItems: number, maxLength: number): string[] {
  return uniqueList(raw.split(","), maxItems, maxLength);
}

export function sanitizeProfile(profile: UserProfile): UserProfile {
  return {
    personal: {
      ...profile.personal,
      fullName: normalizeText(profile.personal.fullName, LIMITS.fullName),
      profession: normalizeText(profile.personal.profession, LIMITS.profession),
      city: normalizeText(profile.personal.city, LIMITS.city),
    },
    lifestyle: {
      ...profile.lifestyle,
    },
    preferences: {
      ...profile.preferences,
      likes: uniqueList(profile.preferences.likes, LIMITS.likes, LIMITS.itemLength),
      dislikes: uniqueList(profile.preferences.dislikes, LIMITS.dislikes, LIMITS.itemLength),
      mealStyles: profile.preferences.mealStyles.slice(0, LIMITS.mealStyles),
      mealTypes: profile.preferences.mealTypes.slice(0, LIMITS.mealTypes),
      preferredCuisines: profile.preferences.preferredCuisines.slice(0, LIMITS.cuisines),
      excludedCuisines: profile.preferences.excludedCuisines.slice(0, LIMITS.cuisines),
    },
    constraints: {
      ...profile.constraints,
      allergies: profile.constraints.allergies.slice(0, LIMITS.allergies),
      conditions: profile.constraints.conditions.slice(0, LIMITS.conditions),
      excludedIngredients: uniqueList(
        profile.constraints.excludedIngredients,
        LIMITS.excludedIngredients,
        LIMITS.itemLength,
      ),
      chronicDiseases: profile.constraints.chronicDiseases.slice(0, LIMITS.chronicDiseases),
      medications: normalizeText(profile.constraints.medications, LIMITS.medications),
    },
  };
}
