import { ApiError } from "@/lib/api";

type ErrorContext =
  | "auth.login"
  | "auth.register"
  | "profile.load"
  | "profile.submit"
  | "recommendations.load"
  | "recommendations.explain";

const DEFAULT_MESSAGES: Record<ErrorContext, string> = {
  "auth.login": "Impossible de te connecter pour le moment.",
  "auth.register": "Impossible de creer le compte pour le moment.",
  "profile.load": "Impossible de charger le profil.",
  "profile.submit": "Impossible d'enregistrer le profil pour le moment.",
  "recommendations.load": "Impossible de charger les recommandations.",
  "recommendations.explain": "Impossible de charger l'explication de ce repas.",
};

export function getSafeErrorMessage(error: unknown, context: ErrorContext): string {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      switch (context) {
        case "auth.login":
          return "Les identifiants sont invalides ou la session n'est pas disponible.";
        case "profile.load":
        case "profile.submit":
        case "recommendations.load":
        case "recommendations.explain":
          return "Connecte-toi pour continuer.";
        default:
          return DEFAULT_MESSAGES[context];
      }
    }

    if (error.status === 403) {
      return "Cette action n'est pas autorisee.";
    }

    if (error.status === 404) {
      if (context === "profile.load") {
        return "Aucun profil enregistre pour le moment.";
      }

      return DEFAULT_MESSAGES[context];
    }

    if (error.status === 409) {
      return DEFAULT_MESSAGES[context];
    }

    if (error.status === 429) {
      return "Trop de tentatives ou de requetes. Reessaie dans un instant.";
    }

    if (error.status >= 400 && error.status < 500) {
      if (context === "profile.submit") {
        return "Certaines informations du profil sont invalides ou incompletes.";
      }

      return DEFAULT_MESSAGES[context];
    }

    return DEFAULT_MESSAGES[context];
  }

  return DEFAULT_MESSAGES[context];
}
