# AGENTS.md

## Vue d'ensemble

NutriMatch est une application composée d'un backend Go (API REST, clean architecture, sécurité avancée) et d'un frontend Next.js (App Router, formulaires multi-étapes, stockage local, UI modulaire). L'objectif est de générer des recommandations de repas personnalisées selon le profil nutritionnel de l'utilisateur, en intégrant préférences, mode de vie et contraintes santé.

---

## Architecture & Flux
- **Frontend** :
  - Pages principales dans `frontend/src/app/` :
    - `onboarding/page.tsx` (formulaire profil multi-étapes, progression contrôlée par `useProfileForm`)
    - `results/page.tsx` (affichage recommandations, données simulées via `mock-data.ts`)
    - `profile/page.tsx` (récapitulatif profil, calculs santé via `health.ts`)
  - Données profil persistées dans `localStorage` (`nutrimatch-profile`).
  - Validation stricte par étape (`lib/validation.ts`), blocage de progression si erreurs.
  - UI : composants réutilisables dans `components/ui/`, styles globaux préfixés `nm-` (`globals.css`).
  - Types métier centralisés dans `lib/types.ts`.
  - Appels API via `lib/api.ts` (utilise `NEXT_PUBLIC_API_URL`).

- **Backend** :
  - API REST Gin (`/api/v1/`), endpoints principaux :
    - `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`
    - `POST /profile`, `GET /profile`, `GET /recommendations/:profileId`
  - Architecture modulaire :
    - `internal/clients/` (APIs externes : GoogleAI, Spoonacular)
    - `internal/services/` (logique métier)
    - `internal/repository/gorm/` (accès DB via GORM)
    - `internal/http/handlers/` (handlers API)
    - `internal/http/middleware/` (auth, rate limit, sécurité)
    - `internal/models/` (entités métier)
  - Sécurité : JWT, Argon2id, rate limiting, CORS configurable, logs, validation stricte.
  - Migrations via Goose (`migrations/`).

---

## Workflows & Conventions
- **Frontend** :
  - Scripts : `npm run dev`, `build`, `start`, `lint` (`frontend/package.json`).
  - Alias TypeScript `@/*` → `frontend/src/*` (`tsconfig.json`).
  - Les résultats sont simulés côté client (`mock-data.ts`), l'intégration API attendue (`api.ts`).
  - Les formulaires sont découpés en étapes, chaque étape a son composant (`components/forms/`).
  - Les erreurs de validation bloquent la navigation (`useProfileForm.ts`, `validation.ts`).

- **Backend** :
  - Lancement : `go run ./cmd/api` (voir `backend/README.md`).
  - Tests : `go test ./...`
  - Migrations : voir section Goose dans `backend/README.md`.
  - Configuration via `.env` (voir `.env.example`).
  - Endpoints attendus par le frontend : `POST /api/profile`, `GET /api/recommendations/:profileId`.

---

## Intégrations & Sécurité
- **APIs externes** : Spoonacular (recettes), Google AI Studio (affinage IA).
- **Sécurité** :
  - Authentification JWT, refresh tokens hachés (Argon2id, pepper).
  - Rate limiting, CORS, headers de sécurité, validation stricte des entrées.
  - Voir `doc/stride.md` pour la modélisation des menaces et mesures de sécurité (STRIDE, Zero Trust, isolation IA, audit trail).

---

## Points d'attention pour agents
- **Frontend** :
  - Ne pas supposer un Next.js standard : conventions modifiées, voir `frontend/AGENTS.md`.
  - Les données de profil sont locales tant que l'intégration API n'est pas activée.
  - Les composants UI et formulaires sont découplés, privilégier la réutilisation.
- **Backend** :
  - Respecter la séparation des couches (handlers, services, repository, models).
  - Toute logique sensible (validation, sécurité) doit passer par les middlewares/services dédiés.
- **Sécurité** :
  - Toujours vérifier les flux et points d'intégration selon les recommandations de `doc/stride.md`.
  - Les données sensibles doivent être anonymisées avant tout appel IA.

---

## Fichiers clés à consulter
- `frontend/AGENTS.md` (conventions Next.js)
- `backend/AGENTS.md` (règles d'architecture)
- `doc/stride.md` (sécurité, menaces)
- `doc/nutrimatch.md` (présentation fonctionnelle)
- `backend/README.md`, `frontend/README.md` (workflows)
- `structure.txt` (vue d'ensemble du projet)

