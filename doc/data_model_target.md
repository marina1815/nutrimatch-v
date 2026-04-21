# Data Model Target

Conception cible pour la refonte SQL et vectorielle de `NutriMatch`.

Objectifs :
- separer clairement identite, sante, taxonomie, recommandation et securite
- remplacer les listes JSONB "metier" par des tables relationnelles quand la donnee doit etre filtree, validee ou historisee
- garder JSONB seulement pour les traces, metadata externes et charges peu structurees
- permettre une reinitialisation propre via une migration initiale unique
- preparer une couche vectorielle utile sans exposer inutilement les donnees de sante

## 1. Principes

- La base primaire reste PostgreSQL.
- La couche vectorielle doit etre implantee dans PostgreSQL via `pgvector`.
- Les donnees de sante sensibles restent dans des tables dediees, avec exposition minimale.
- Les valeurs metier "canonique" vivent dans des tables de reference et non dans le code seul.
- Les recommandations doivent etre auditables: on garde le run, les candidats, les raisons de rejet, les raisons d'acceptation et la provenance.
- Les appels IA ne doivent jamais recevoir directement tout le profil brut.

## 2. Schemas proposes

- `identity`
  Gestion des comptes, identites externes, sessions, verification email.
- `catalog`
  Taxonomies metier et mappings vers Spoonacular.
- `health`
  Profil utilisateur, contraintes, objectifs, snapshots nutritionnels derives.
- `recommendation`
  Runs, candidats, explications, cache des resolutions externes.
- `security`
  Audit, evenements de securite, rate limit persistant, secrets metadata.

## 3. Identite

Tables cibles :
- `identity.users`
  `id`, `email`, `password_hash`, `full_name`, `status`, `email_verified`, `created_at`, `updated_at`
- `identity.user_emails`
  emails secondaires si on veut evoluer plus tard
- `identity.external_identities`
  fournisseur OIDC, issuer, subject, dernier login
- `identity.sessions`
  `id`, `user_id`, `auth_method`, `refresh_token_hash`, `csrf_binding_id`, `expires_at`, `idle_expires_at`, `revoked_at`, `ip_hash`, `user_agent_hash`, `created_at`
- `identity.email_verification_tokens`
- `identity.password_reset_tokens`

Notes securite :
- stocker les refresh tokens hashes uniquement
- prevoir expiration absolue + idle timeout
- stocker hash ou version reduite de l'IP et du user-agent si possible

## 4. Catalogue metier

Tables cibles :
- `catalog.ingredients`
  ingredient canonique interne
- `catalog.ingredient_aliases`
  alias FR/EN, synonymes, fautes courantes, forme libre
- `catalog.intolerances`
  intolerance canonique interne
- `catalog.intolerance_aliases`
- `catalog.conditions`
  condition medicale canonique interne
- `catalog.condition_aliases`
- `catalog.medication_patterns`
  patterns medicamenteux relies a des restrictions
- `catalog.meal_styles`
- `catalog.diets`
- `catalog.cuisines`
- `catalog.spoonacular_ingredient_map`
  lien ingredient interne -> ingredient Spoonacular resolu
- `catalog.spoonacular_param_map`
  lien valeur interne -> valeur API externe (`diet`, `intolerances`, `type`, `cuisine`)
- `catalog.medical_rules`
  versionnee, activable, avec severite et contraintes nutritionnelles

Pourquoi :
- aujourd'hui beaucoup de vocabulaire est gere via JSON ou via code
- pour un projet securise, les contraintes dures doivent pouvoir etre verifiees et tracees proprement

## 5. Sante et profil

Tables cibles :
- `health.profiles`
  un profil courant par utilisateur
- `health.profile_personal`
  age, sexe, taille, poids, profession, ville
- `health.profile_lifestyle`
  activite, type de vie, objectif
- `health.profile_preferences`
  donnees libres courantes
- `health.profile_preference_ingredients`
  `profile_id`, `ingredient_id`, `kind` = `like|dislike|exclude`
- `health.profile_meal_styles`
  styles choisis
- `health.profile_conditions`
  conditions declarees
- `health.profile_intolerances`
  intolerances declarees
- `health.profile_medications`
  medicaments libres normalises et resolves si possible
- `health.profile_snapshots`
  version du profil a chaque soumission importante
- `health.nutrition_profiles`
  snapshot derive numerique: BMI, BMR, objectifs calories/macros, sodium/sucre, restrictions derivees

Choix :
- separer les preferences ingrediants/intolerances/conditions permet validation forte, indexation et audit
- garder un snapshot derive evite de recalculer partout et permet de relire l'etat au moment d'une recommandation

## 6. Recommendation et traces

Tables cibles :
- `recommendation.runs`
  run global
- `recommendation.run_inputs`
  signature du filtre, resume des sources, modele utilise, mode degrade ou non
- `recommendation.candidates`
  recettes candidates avec nutrition, score, rang final
- `recommendation.candidate_filters`
  une ligne par decision de filtrage dur
- `recommendation.candidate_scores`
  breakdown du score deterministe
- `recommendation.explanations`
  explication finale rendue au frontend
- `recommendation.external_recipe_cache`
  copie partielle et TTL des recettes/details Spoonacular
- `recommendation.ingredient_resolution_cache`
  resultat de resolution d'un ingredient libre vers ingredient canonique / Spoonacular

Pourquoi :
- eviter les gros blobs JSON pour les decisions critiques
- garder des tables fines pour investiguer les rejections et prouver le respect des regles sante

## 7. Securite et audit

Tables cibles :
- `security.audit_events`
  evenement horodate, immutable en pratique applicative
- `security.auth_failures`
  suivi anti-bruteforce et anti-enumeration
- `security.rate_limit_buckets`
  utile si on veut sortir du full in-memory multi-instance
- `security.secret_versions`
  metadata de rotation, sans secret brut

Exigences :
- pas de donnees sante brutes dans les logs
- request id, user id et session id relies a chaque action sensible
- traces d'acces aux recommandations et au profil

## 8. Couche vectorielle

Technologie cible :
- PostgreSQL + extension `vector`

Tables cibles :
- `recommendation.profile_embeddings`
  embedding d'un profil derive et minimise
- `recommendation.recipe_embeddings`
  embedding d'une recette candidate ou d'un resume de recette
- `recommendation.preference_embeddings`
  embedding d'agregats de preferences anonymises si utile

Regles :
- ne jamais vectoriser le profil brut complet avec texte libre sensible
- vectoriser un resume derive, minimise et nettoye
- versionner les embeddings (`embedding_version`)
- prevoir `source_hash` pour savoir quand recalculer

Cas d'usage autorises :
- expansion de recettes similaires
- rapprochement semantique entre styles/ingredients et recettes
- pas de decision sante finale fondee uniquement sur la similarite vectorielle

## 9. Index et contraintes

A prevoir des le depart :
- index uniques sur les valeurs canoniques de taxonomie
- index composites sur les tables de liaison profil/taxonomie
- index sur `recommendation.runs(profile_id, created_at desc)`
- index sur `recommendation.candidates(run_id, final_rank)`
- index sur `security.audit_events(user_id, occurred_at desc)`
- contraintes `CHECK` sur enumerations et bornes nutritionnelles
- FKs fortes avec `ON DELETE CASCADE` seulement quand le domaine le justifie

## 10. Migration cible

Strategie recommandee :
1. creer une nouvelle migration initiale propre
2. charger les tables de reference (`catalog.*`)
3. brancher progressivement services/repositories sur le nouveau schema
4. supprimer les JSONB metier non necessaires une fois la refonte terminee

## 11. Decisions ouvertes

- faut-il garder un seul profil courant ou versionner pleinement chaque soumission ?
- quelle part de la taxonomie ingredients doit etre locale vs derivee de Spoonacular ?
- quel niveau de cache externe garder pour limiter les couts API ?
- quel sous-ensemble de donnees anonymisees peut etre envoye au reranking IA ?
