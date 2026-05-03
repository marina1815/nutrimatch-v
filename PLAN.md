# PLAN.md

Plan directeur actuel de NutriMatch.

## Objectif

NutriMatch recommande des repas personnalises depuis un catalogue local complet. Le backend reste l'autorite de securite: allergies, allergies croisees, exclusions, maladies, medicaments et regles metier sont appliquees avant toute intervention IA.

L'IA ne peut jamais ajouter une recette hors catalogue, valider une recette dangereuse, changer un score ou contourner une contrainte. Dans le flux quotidien, elle peut choisir et expliquer 20 recettes uniquement parmi un pool deja juge sur par le backend. Si sa sortie est lente, invalide ou indisponible, le backend utilise un tirage deterministe pondere.

## Architecture Cible

- Frontend Next.js en francais, pages publiques `/`, `/login`, `/signin`, `/register`, `/signup`, `/signout`.
- Backend Go/Gin en clean architecture avec handlers, services, repositories GORM et modeles separes.
- PostgreSQL avec schemas `identity`, `catalog`, `health`, `recommendation`, `security` et extension `pgvector`.
- Redis pour sessions/rate-limit/cache selon configuration.
- Docker compose complet avec ports par defaut `3000`, `8080`, `5432`, `6379`.

## Catalogue Local

- Source primaire et obligatoire: fichiers locaux importes dans `catalog.*`.
- Recettes, ingredients, allergies et allergies croisees sont normalises avec trim, dropna logique et deduplication.
- Les macros locales sont marquees comme estimees.
- Les decisions de securite dures reposent sur ingredients, allergies, exclusions, maladies, medicaments et regles medicales, pas sur l'IA.
- L'ancienne API recettes externe est retiree du code applicatif et de la configuration.

## Recommandations Quotidiennes

- `GET /api/v1/recommendations/:profileId` retourne le set actif si sa fenetre 24h est encore valide.
- Si aucun set actif n'existe, le backend evalue le catalogue local complet ou un pool large configurable.
- Le backend applique les hard constraints avant le scoring.
- Soft constraints: aliments aimes/non aimes, cuisines, types de repas, objectif, similarite.
- Le set contient jusqu'a 20 recettes sures.
- Les recettes du set precedent sont evitees si assez de recettes sures existent.
- Une recette choisie est exclue des nouvelles suggestions pendant 7 jours.
- Le frontend affiche un compte a rebours avant la prochaine selection.

## IA

- Appel batch unique par fenetre 24h pour les 20 explications.
- Entree IA: IDs autorises, titre, ingredients et faits nutritionnels minimises.
- Sortie IA acceptee: tableau JSON d'objets `{ "mealId": "...", "explanation": "..." }`.
- Sortie IA refusee: ID inconnu, score, ordre, verdict de securite, recette inventee, champ non autorise, doublon, explication vide.
- `POST /api/v1/recommendations/:profileId/meals/:mealId/choose` marque une recette choisie et peut demander un guide de preparation estime.
- Les substitutions IA sont revalidees contre allergies, maladies, exclusions et medicaments avant stockage.

## Securite

- Auth locale avec sessions, refresh tokens haches, rotation et revocation.
- MFA optionnel: TOTP et passkeys WebAuthn.
- CSRF lie a la session pour les actions cookie.
- CORS/origines de confiance explicites.
- Donnees sante sensibles chiffrees au repos.
- Audit trail hash-chain pour evenements securite et metier.
- Rate limiting partage via repository/Redis selon configuration.
- Controle d'acces par propriete utilisateur et politique applicative.

## Frontend

- Interface en francais.
- Onboarding charge le profil existant et la taxonomie backend.
- Les ingredients libres passent par l'autocompletion catalogue.
- Les pages protegees redirigent vers `/login` sans session.
- Les cartes resultats affichent uniquement nom, ingredients, explication et bouton "Choisir cette recette".
- Pas de donnees sante persistantes en `localStorage`.

## Validation

Commandes attendues avant point de sauvegarde:

```powershell
cd backend
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

```powershell
cd frontend
npm run lint
npm run build
npm audit --omit=dev
```

Smoke Docker:

```powershell
Copy-Item .env.docker.example .env
docker compose up --build -d
Invoke-RestMethod http://localhost:8080/api/v1/health
docker compose down
```

## Etat

- Catalogue local: en place avec 814 recettes seed.
- Suppression de l'ancienne API recettes externe: en place cote code/config, docs alignees dans cette passe.
- Suggestions 24h: en place.
- Exclusion recette choisie 7 jours: en place.
- IA batch safe-pool avec fallback: en place.
- Frontend resultats quotidien: en place.
- Docker compose full-stack: en place.
- Tests finaux: a relancer apres chaque modification.
