"use client";

import { useEffect, useState } from "react";
import { UserProfile } from "@/lib/types";
import { validateStep } from "@/lib/validation";

const STORAGE_KEY = "nutrimatch-profile";

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
  },
  preferences: {
    likes: [],
    dislikes: [],
    mealStyles: [],
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
  const [data, setData] = useState<UserProfile>(defaultProfile);
  const [errors, setErrors] = useState<Record<string, any>>({});

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);

    if (saved) {
      try {
        const parsed = JSON.parse(saved) as Partial<UserProfile>;
        setData(mergeWithDefaultProfile(parsed));
      } catch {
        setData(defaultProfile);
      }
    }
  }, []);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  }, [data]);

  const next = () => {
    const stepErrors = validateStep(step, data);

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
    localStorage.removeItem(STORAGE_KEY);
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