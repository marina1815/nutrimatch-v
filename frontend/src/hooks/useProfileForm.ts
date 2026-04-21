"use client";

import { useEffect, useState } from "react";
import { clearDraftProfile, getDraftProfile, setDraftProfile } from "@/lib/session";
import { sanitizeProfile } from "@/lib/profile-normalization";
import { UserProfile } from "@/lib/types";
import { ProfileErrors, validateStep } from "@/lib/validation";

const defaultProfile: UserProfile = {
  personal: {
    fullName: "",
    age: "",
    sex: "",
    weight: "",
    height: "",
    profession: "",
    city: "",
  },
  lifestyle: {
    activityLevel: "",
    lifestyleType: "",
    goal: "",
    maxReadyTime: 45,
  },
  preferences: {
    likes: [],
    dislikes: [],
    mealStyles: [],
    mealTypes: [],
    preferredCuisines: [],
    excludedCuisines: [],
    mealsPerDay: "",
  },
  constraints: {
    allergies: [],
    conditions: [],
    excludedIngredients: [],
    hasChronicDisease: false,
    chronicDiseases: [],
    takesMedication: false,
    medications: "",
  },
};

function mergeWithDefaultProfile(saved: Partial<UserProfile>): UserProfile {
  return {
    personal: {
      ...defaultProfile.personal,
      ...saved.personal,
    },
    lifestyle: {
      ...defaultProfile.lifestyle,
      ...saved.lifestyle,
    },
    preferences: {
      ...defaultProfile.preferences,
      ...saved.preferences,
      likes: saved.preferences?.likes ?? [],
      dislikes: saved.preferences?.dislikes ?? [],
      mealStyles: saved.preferences?.mealStyles ?? [],
      mealTypes: saved.preferences?.mealTypes ?? [],
      preferredCuisines: saved.preferences?.preferredCuisines ?? [],
      excludedCuisines: saved.preferences?.excludedCuisines ?? [],
      mealsPerDay: saved.preferences?.mealsPerDay ?? "",
    },
    constraints: {
      ...defaultProfile.constraints,
      ...saved.constraints,
      allergies: saved.constraints?.allergies ?? [],
      conditions: saved.constraints?.conditions ?? [],
      excludedIngredients: saved.constraints?.excludedIngredients ?? [],
      chronicDiseases: saved.constraints?.chronicDiseases ?? [],
      medications: saved.constraints?.medications ?? "",
      hasChronicDisease: saved.constraints?.hasChronicDisease ?? false,
      takesMedication: saved.constraints?.takesMedication ?? false,
    },
  };
}

export function useProfileForm() {
  const [step, setStep] = useState(0);
  const [data, setData] = useState<UserProfile>(() => {
    const saved = getDraftProfile();
    return saved ? mergeWithDefaultProfile(saved) : defaultProfile;
  });
  const [errors, setErrors] = useState<ProfileErrors>({});

  useEffect(() => {
    setDraftProfile(data);
  }, [data]);

  const next = () => {
    const sanitized = sanitizeProfile(data);
    setData(sanitized);
    const stepErrors = validateStep(step, sanitized);

    if (Object.keys(stepErrors).length > 0) {
      setErrors(stepErrors);
      return false;
    }

    setErrors({});
    setStep((prev) => Math.min(prev + 1, 3));
    return true;
  };

  const back = () => setStep((prev) => Math.max(prev - 1, 0));

  const reset = () => {
    setData(defaultProfile);
    setErrors({});
    setStep(0);
    clearDraftProfile();
  };

  return {
    step,
    data,
    setData,
    errors,
    next,
    back,
    reset,
  };
}
