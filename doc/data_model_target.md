# Modele De Donnees Cible

## Schemas

- `identity`: utilisateurs, sessions, identites externes.
- `health`: profil, mode de vie, preferences, contraintes, profil nutritionnel calcule.
- `catalog`: recettes locales, ingredients, allergies, allergies croisees, taxonomies, regles medicales.
- `recommendation`: runs, candidats, sets quotidiens, repas de set, choix utilisateur.
- `security`: audit trail, rate limit, auth failures.

## Catalogue

Tables principales:

- `catalog.local_recipes`;
- `catalog.local_ingredients`;
- `catalog.local_recipe_ingredients`;
- `catalog.local_allergies`;
- `catalog.local_ingredient_allergies`;
- `catalog.local_cross_allergies`;
- `catalog.medical_rules`.

Les donnees viennent des fichiers Excel fournis et sont normalisees avant seed.

## Recommandations

Tables principales:

- `recommendation.runs`: trace d'une generation.
- `recommendation.candidates`: decisions par recette candidate.
- `recommendation.daily_sets`: fenetre 24h active.
- `recommendation.daily_set_meals`: les recettes affichees dans le set.
- `recommendation.recipe_choices`: recette choisie et exclusion 7 jours.

## Vectoriel

`pgvector` est utilise comme signal positif secondaire:

- similarite profil/profil;
- similarite recette/intention;
- versionnement des embeddings;
- aucune capacite a contourner les hard constraints.

## Retention

- Sessions expirees: supprimees selon politique.
- Rate-limit buckets: nettoyes apres expiration.
- Traces de recommandations: retenues selon configuration.
- Audit security: append-only, hors purge applicative standard.

## Donnees Sensibles

- Medicaments et donnees sante sensibles: chiffrement applicatif au repos.
- Index aveugles: cle dediee quand recherche/pseudonymisation necessaire.
- Logs et traces: pas de secret, pas de payload sante brut inutile.
