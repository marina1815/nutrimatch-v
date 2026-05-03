import {
  ACTIVITY_OPTIONS,
  CHRONIC_DISEASE_OPTIONS,
  COMMON_ALLERGIES,
  COMMON_CONDITIONS,
  GOAL_OPTIONS,
  LIFESTYLE_OPTIONS,
  SEX_OPTIONS,
} from "@/lib/constants";
import { getIngredientLabel } from "@/lib/ingredient-labels";
import { RecommendationResponse } from "@/lib/types";

type Option = { value: string; label: string };

function optionLabel(options: ReadonlyArray<Option>, value: string): string {
  return options.find((option) => option.value === value)?.label || value;
}

function listLabels(values: string[], mapper: (value: string) => string): string {
  const labels = values.map(mapper).filter(Boolean);
  return labels.length > 0 ? labels.join(", ") : "-";
}

const LOCAL_ALLERGY_TERMS: Record<string, string> = {
  "alliaceae": "Alliacées",
  "alliums": "Ail, oignon et alliacées",
  "alpha gal": "Alpha-gal",
  "aquagenic": "Aquagénique",
  "aspirin": "Aspirine",
  "asteraceae": "Astéracées",
  "banana": "Banane",
  "beer": "Bière",
  "brassicaceae": "Brassicacées",
  "broccoli": "Brocoli",
  "celery": "Céleri",
  "cereal": "Céréales",
  "citrus": "Agrumes",
  "coconut": "Noix de coco",
  "corn": "Maïs",
  "cruciferous": "Crucifères",
  "cucumisin": "Cucumisine",
  "cucurbitaceae": "Cucurbitacées",
  "egg": "Oeufs",
  "exotic fruit": "Fruits exotiques",
  "fish": "Poisson",
  "fructan": "Fructanes",
  "fructose": "Fructose",
  "gluten": "Gluten",
  "histamine": "Histamine",
  "honey bee hive": "Produits de la ruche",
  "immediate type": "Type immédiat",
  "insoluble fiber": "Fibres insolubles",
  "insulin": "Insuline",
  "lactose": "Lactose",
  "lamiaceae": "Lamiacées",
  "latex": "Latex",
  "legume or fabaceae": "Légumineuses",
  "lesser oral": "Syndrome oral léger",
  "lily family": "Famille des liliacées",
  "lipid transfer protein": "Protéine de transfert lipidique",
  "lupin": "Lupin",
  "milk": "Lait",
  "mint": "Menthe",
  "mollusk": "Mollusques",
  "mushroom": "Champignons",
  "mustard": "Moutarde",
  "myristicaceae": "Myristicacées",
  "nut": "Fruits à coque",
  "oral allergy syndrome oas": "Syndrome d'allergie orale",
  "peanut": "Arachides",
  "pepper": "Poivre et piment",
  "pollen food allergy syndrome pfas": "Syndrome pollen-aliment",
  "polyol": "Polyols",
  "potato": "Pomme de terre",
  "poultry": "Volaille",
  "profilin": "Profiline",
  "rice": "Riz",
  "rosaceae": "Rosacées",
  "rutaceae or citrus": "Rutacées ou agrumes",
  "salicylate": "Salicylates",
  "seed": "Graines",
  "sesame seed": "Sésame",
  "shellfish": "Fruits de mer",
  "solanaceae": "Solanacées",
  "soy": "Soja",
  "spice": "Épices",
  "stone fruit": "Fruits à noyau",
  "sugar": "Sucre",
  "umbelliferae": "Apiacées",
  "wheat and triticale": "Blé et triticale",
};

function normalizeCatalogLabel(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .toLowerCase();
}

function sentenceCase(value: string): string {
  const cleaned = value.trim();
  return cleaned ? cleaned[0].toUpperCase() + cleaned.slice(1) : cleaned;
}

function cleanAllergySource(value: string): string {
  return normalizeCatalogLabel(value)
    .replace(/\b(non allergy|allergy|intolerance|syndrome|urticaria)\b/g, " ")
    .replace(/\boas\b/g, "oas")
    .replace(/\bpfas\b/g, "pfas")
    .replace(/\s+/g, " ")
    .trim();
}

function isTechnicalAllergyLabel(value: string): boolean {
  const normalized = normalizeCatalogLabel(value);
  return (
    normalized.includes("cross between") ||
    normalized.includes("cross allergy") ||
    normalized.includes("pollen food") ||
    normalized.includes("pfas") ||
    (normalized.includes("oral") && normalized.includes("oas")) ||
    normalized.includes("non allergy")
  );
}

function translateLocalAllergyTerm(value: string): string {
  const normalized = cleanAllergySource(value);
  return LOCAL_ALLERGY_TERMS[normalized] || sentenceCase(normalized);
}

export function labelAllergyValue(value: string, fallback?: string): string {
  const common = optionLabel(COMMON_ALLERGIES, value);
  if (common !== value) {
    return common;
  }

  const source = fallback || value;
  if (isTechnicalAllergyLabel(value) || isTechnicalAllergyLabel(source)) {
    return "-";
  }
  const normalized = cleanAllergySource(source);
  if (!normalized || normalized === "non") {
    return "-";
  }
  return translateLocalAllergyTerm(normalized);
}

export function labelActivity(value: string): string {
  return optionLabel(ACTIVITY_OPTIONS, value);
}

export function labelLifestyle(value: string): string {
  return optionLabel(LIFESTYLE_OPTIONS, value);
}

export function labelGoal(value: string): string {
  return optionLabel(GOAL_OPTIONS, value);
}

export function labelSex(value: string): string {
  return optionLabel(SEX_OPTIONS, value);
}

export function labelAllergies(values: string[]): string {
  return listLabels(values, (value) => labelAllergyValue(value));
}

export function labelConditions(values: string[]): string {
  return listLabels(values, (value) => {
    const common = optionLabel(COMMON_CONDITIONS, value);
    return common === value ? optionLabel(CHRONIC_DISEASE_OPTIONS, value) : common;
  });
}

export function labelIngredients(values: string[]): string {
  return listLabels(values, getIngredientLabel);
}

export function labelSelectionMode(value?: string): string {
  switch (value) {
    case "backend_random_ai_validated":
      return "Tirage backend valide par IA";
    case "backend_random_ai_partial":
      return "Tirage backend partiellement valide par IA";
    case "backend_random_ai_unavailable":
      return "Tirage backend sans IA";
    case "ai_safe_pool_selection":
      return "Sélection IA contrôlée";
    case "deterministic_weighted_fallback":
      return "Sélection déterministe sécurisée";
    case "none":
      return "Aucune recette disponible";
    default:
      return value || "-";
  }
}

export function labelAIStatus(response: RecommendationResponse): string {
  return response.aiExplanationApplied ? "Explications IA prêtes" : "Explications IA indisponibles";
}

export function labelAIReason(value?: string): string {
  switch (value) {
    case "":
    case undefined:
      return "";
    case "ai_client_unavailable":
    case "ai_key_missing":
      return "Clé IA manquante dans l'environnement backend.";
    case "ai_network_unreachable":
      return "Connexion internet indisponible, Wi-Fi coupé ou connexion au service IA refusée.";
    case "ai_dns_error":
      return "Le domaine du service IA est impossible à résoudre.";
    case "ai_timeout":
    case "context_deadline_exceeded":
      return "Le délai de réponse IA a été dépassé.";
    case "ai_auth_failed":
      return "Clé IA invalide ou refusée par le fournisseur.";
    case "ai_rate_limited":
      return "Quota ou rate limit IA atteint. Réessaie plus tard.";
    case "ai_upstream_unavailable":
      return "Service IA temporairement indisponible.";
    case "ai_bad_request":
      return "Configuration ou payload IA refusé par le fournisseur.";
    case "ai_empty_response":
      return "Le service IA a répondu vide.";
    case "ai_invalid_response":
      return "La réponse IA est illisible ou invalide.";
    case "ai_generation_failed":
      return "La génération IA a échoué.";
    case "no_safe_candidates":
      return "Aucune recette sûre disponible après les filtres santé.";
    case "no_accepted_meals":
      return "Aucune recette acceptée par les contraintes du profil.";
    default:
      if (value.startsWith("ai_output_forbidden_field_")) {
        return "La réponse IA contenait un champ interdit et a été ignorée.";
      }
      return value.replace(/_/g, " ");
  }
}

export function isAIRetryableReason(value?: string): boolean {
  switch (value) {
    case "ai_network_unreachable":
    case "ai_dns_error":
    case "ai_timeout":
    case "context_deadline_exceeded":
    case "ai_rate_limited":
    case "ai_upstream_unavailable":
    case "ai_empty_response":
    case "ai_invalid_response":
      return true;
    default:
      return !!value?.startsWith("ai_output_");
  }
}

export function labelBmiCategory(value: string): string {
  switch (value) {
    case "underweight":
      return "Insuffisance pondérale";
    case "normal":
      return "Corpulence normale";
    case "overweight":
      return "Surpoids";
    case "obese":
      return "Obésité";
    default:
      return value;
  }
}

export function labelAuthMethod(value: string): string {
  switch (value) {
    case "local":
      return "Mot de passe";
    case "oidc":
      return "Connexion externe";
    case "mfa":
      return "MFA";
    default:
      return value || "Session";
  }
}
