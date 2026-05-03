import { HealthMetrics, UserProfile } from "@/lib/types";

export function calculateBMI(weight: number, heightCm: number): number {
  const heightM = heightCm / 100;
  return Number((weight / (heightM * heightM)).toFixed(1));
}

export function getBMICategory(bmi: number): string {
  if (bmi < 18.5) return "Insuffisance pondérale";
  if (bmi < 25) return "Poids normal";
  if (bmi < 30) return "Surpoids";
  return "Obésité";
}

export function calculateBMR(profile: UserProfile): number {
  const weight = Number(profile.personal.weight);
  const height = Number(profile.personal.height);
  const age = Number(profile.personal.age);

  if (!weight || !height || !age || !profile.personal.sex) return 0;

  if (profile.personal.sex === "male") {
    return Math.round(10 * weight + 6.25 * height - 5 * age + 5);
  }

  return Math.round(10 * weight + 6.25 * height - 5 * age - 161);
}

export function getActivityMultiplier(level: UserProfile["lifestyle"]["activityLevel"]): number {
  switch (level) {
    case "sedentary":
      return 1.2;
    case "light":
      return 1.375;
    case "moderate":
      return 1.55;
    case "active":
      return 1.725;
    default:
      return 1.2;
  }
}

export function calculateEstimatedCalories(profile: UserProfile): number {
  const bmr = calculateBMR(profile);
  const maintenance = Math.round(bmr * getActivityMultiplier(profile.lifestyle.activityLevel));

  switch (profile.lifestyle.goal) {
    case "weight_loss":
      return maintenance - 300;
    case "muscle_gain":
      return maintenance + 250;
    case "weight_maintenance":
      return maintenance;
    case "energy_maintenance":
      return maintenance;
    case "medical_diet":
      return maintenance;
    default:
      return maintenance;
  }
}

export function getHealthMetrics(profile: UserProfile): HealthMetrics {
  const bmi = calculateBMI(Number(profile.personal.weight), Number(profile.personal.height));
  const bmiCategory = getBMICategory(bmi);
  const bmr = calculateBMR(profile);
  const estimatedCalories = calculateEstimatedCalories(profile);

  return {
    bmi,
    bmiCategory,
    bmr,
    estimatedCalories,
  };
}