import { ApiError } from "@/lib/api";

type ErrorContext =
  | "auth.login"
  | "auth.register"
  | "profile.load"
  | "profile.submit"
  | "profile.security.load"
  | "profile.security.action"
  | "profile.security.password"
  | "profile.security.totp.setup"
  | "profile.security.totp.confirm"
  | "profile.security.totp.disable"
  | "profile.security.passkey.register"
  | "profile.security.passkey.verify"
  | "recommendations.load"
  | "recommendations.explain"
  | "recommendations.choose";

const DEFAULT_MESSAGES: Record<ErrorContext, string> = {
  "auth.login": "Impossible de te connecter pour le moment.",
  "auth.register": "Impossible de créer le compte pour le moment.",
  "profile.load": "Impossible de charger le profil.",
  "profile.submit": "Impossible d'enregistrer le profil pour le moment.",
  "profile.security.load": "Impossible de charger les paramètres de sécurité.",
  "profile.security.action": "Impossible d'appliquer cette action de sécurité pour le moment.",
  "profile.security.password": "Impossible de modifier le mot de passe pour le moment.",
  "profile.security.totp.setup": "Impossible de démarrer la configuration authenticator.",
  "profile.security.totp.confirm": "Impossible de confirmer le code authenticator.",
  "profile.security.totp.disable": "Impossible de désactiver authenticator pour le moment.",
  "profile.security.passkey.register": "Impossible d'ajouter cette clé d'accès pour le moment.",
  "profile.security.passkey.verify": "Impossible de vérifier cette clé d'accès pour le moment.",
  "recommendations.load": "Impossible de charger les recommandations.",
  "recommendations.explain": "Impossible de charger l'explication de ce repas.",
  "recommendations.choose": "Impossible de choisir cette recette pour le moment.",
};

export function getSafeErrorMessage(error: unknown, context: ErrorContext): string {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      switch (context) {
        case "auth.login":
          return "Les identifiants sont invalides ou la session n'est pas disponible.";
        case "profile.load":
        case "profile.submit":
        case "profile.security.load":
        case "profile.security.action":
        case "profile.security.password":
        case "profile.security.totp.setup":
        case "profile.security.totp.confirm":
        case "profile.security.totp.disable":
        case "profile.security.passkey.register":
        case "profile.security.passkey.verify":
        case "recommendations.load":
        case "recommendations.explain":
        case "recommendations.choose":
          return "Connecte-toi pour continuer.";
        default:
          return DEFAULT_MESSAGES[context];
      }
    }

    if (error.status === 403) {
      return "Cette action n'est pas autorisée.";
    }

    if (error.status === 404) {
      if (context === "profile.load") {
        return "Aucun profil enregistré pour le moment.";
      }

      return DEFAULT_MESSAGES[context];
    }

    if (error.status === 409) {
      return DEFAULT_MESSAGES[context];
    }

    if (error.status === 429) {
      return "Trop de tentatives ou de requêtes. Réessaie dans un instant.";
    }

    if (error.status >= 400 && error.status < 500) {
      if (context === "profile.submit") {
        return "Certaines informations du profil sont invalides ou incomplètes.";
      }

      return DEFAULT_MESSAGES[context];
    }

    return DEFAULT_MESSAGES[context];
  }

  return DEFAULT_MESSAGES[context];
}
