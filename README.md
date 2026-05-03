# NutriMatch - Guide de lancement et documentation technique

NutriMatch est une application web de recommandations de repas personnalisees. Elle combine un backend Go, un frontend Next.js, PostgreSQL avec pgvector et Redis. Le catalogue de recettes est local et embarque dans l'application; aucune API externe de recettes n'est necessaire au fonctionnement principal.

L'IA Google est optionnelle. Elle intervient uniquement sur des recettes deja validees par le backend afin de les verifier et de produire des explications. Elle ne peut pas contourner les contraintes de securite nutritionnelle appliquees cote backend.

## Sommaire

- [Stack technique](#stack-technique)
- [Architecture](#architecture)
- [Fonctionnement metier](#fonctionnement-metier)
- [Prerequis](#prerequis)
- [Lancement Docker recommande](#lancement-docker-recommande)
- [Variables d'environnement](#variables-denvironnement)
- [Base de donnees](#base-de-donnees)
- [Flux de test manuel](#flux-de-test-manuel)
- [Developpement hors Docker](#developpement-hors-docker)
- [Tests et verification](#tests-et-verification)
- [Depannage](#depannage)
- [Securite](#securite)
- [Distribution du projet](#distribution-du-projet)

## Stack technique

| Couche | Technologie |
|---|---|
| Frontend | Next.js App Router, React, TypeScript |
| Backend | Go, Gin, GORM |
| Base de donnees | PostgreSQL avec extension pgvector |
| Cache/session/rate limit | Redis |
| Migrations | Goose |
| Authentification | JWT access token en memoire, refresh token en cookie HttpOnly |
| MFA | TOTP et Passkeys WebAuthn |
| IA optionnelle | Google AI Studio / Gemini |
| Conteneurisation | Docker Compose |

## Architecture

Le projet est organise autour d'une separation claire entre interface, logique metier, stockage et securite.

```text
NutriMatch/
+-- docker-compose.yml
+-- .env.docker.example
+-- docker/
|   +-- postgres/init-app-role.sh
+-- backend/
|   +-- cmd/api/                 Point d'entree API Go
|   +-- internal/
|   |   +-- clients/             Clients externes optionnels, dont Google AI
|   |   +-- catalog/             Taxonomie, labels, tags et compatibilite sante
|   |   +-- http/                Routes, handlers, middlewares
|   |   +-- localdata/           Catalogue local embarque
|   |   +-- models/              Modeles domaine
|   |   +-- repository/          Interfaces et implementations GORM/Redis
|   |   +-- security/            CSRF, chiffrement, audit, quotas
|   |   +-- services/            Logique applicative
|   +-- migrations/              Schemas SQL Goose
|   +-- Dockerfile
|   +-- go.mod
|   +-- go.sum
+-- frontend/
    +-- public/
    +-- src/
    |   +-- app/                 Pages Next.js
    |   +-- components/          Formulaires, UI, cartes de resultats
    |   +-- lib/                 Client API, session, validation, labels
    +-- Dockerfile
    +-- package.json
    +-- package-lock.json
```

### Schemas PostgreSQL

| Schema | Role |
|---|---|
| `identity` | Utilisateurs, sessions, identites federes, MFA |
| `health` | Profil nutritionnel, contraintes, donnees sante chiffrees |
| `catalog` | Recettes, ingredients, allergies, compatibilites medicales |
| `recommendation` | Sets quotidiens, repas proposes, choix utilisateur, traces |
| `security` | Audit trail append-only et evenements securite |
| `public` | Table Goose et extension pgvector |

## Fonctionnement metier

1. L'utilisateur cree un compte puis remplit son profil nutritionnel.
2. Le backend calcule un profil nutritionnel derive: objectifs, contraintes, exclusions, limites de securite.
3. Le moteur parcourt le catalogue local complet.
4. Les hard constraints sont appliquees en priorite: allergies, allergies croisees, ingredients exclus, maladies, medicaments et regles nutritionnelles.
5. Les soft constraints influencent seulement le score: aliments apprecies, aliments moins apprecies, objectif et similarite.
6. Le backend construit un pool de recettes sures.
7. Vingt recettes sont selectionnees pour une fenetre de 24 heures.
8. Si l'IA est disponible, elle valide ces recettes deja sures et genere les explications.
9. Si l'IA est indisponible, les recettes backend-safe restent disponibles sans explication inventee.
10. Une recette choisie est mise en cache et exclue des futures suggestions pendant 7 jours.

Les contraintes dures ne sont jamais relachees, meme si moins de 20 recettes sont disponibles.

## Prerequis

### Lancement recommande

- Docker Desktop
- Docker Compose v2
- Un terminal PowerShell, Windows Terminal, bash ou equivalent

### Developpement hors Docker

- Go compatible avec `backend/go.mod`
- Node.js compatible avec le frontend Next.js
- npm
- PostgreSQL avec pgvector
- Redis
- Goose

Le lancement Docker est recommande car il fournit directement PostgreSQL avec pgvector, Redis, les migrations, l'API et le frontend.

## Lancement Docker recommande

Depuis le dossier racine du projet, c'est-a-dire le dossier qui contient `docker-compose.yml`.

### Windows PowerShell

```powershell
Copy-Item .env.docker.example .env
docker compose up --build -d
```

### Linux/macOS

```bash
cp .env.docker.example .env
docker compose up --build -d
```

### Verification

```powershell
docker compose ps
Invoke-RestMethod http://localhost:8080/api/v1/health
```

Application web:

```text
http://localhost:3000
```

API:

```text
http://localhost:8080/api/v1/health
```

Le service `migrate` execute automatiquement les migrations avant le demarrage de l'API.

## Variables d'environnement

Pour Docker, seul le fichier `.env` a la racine est lu par `docker-compose.yml`. Il doit etre cree depuis `.env.docker.example`.

| Fichier | Usage |
|---|---|
| `.env.docker.example` | Modele portable pour Docker Compose. A copier en `.env` a la racine. |
| `.env` | Configuration locale reelle de Docker Compose. Non versionne. |
| `backend/.env.example` | Modele optionnel pour lancer uniquement le backend hors Docker. |
| `frontend/.env.example` | Modele optionnel pour lancer uniquement le frontend hors Docker. |

Les fichiers reels `.env`, `backend/.env` et `frontend/.env.local` ne doivent pas etre versionnes car ils peuvent contenir des secrets.

### Variables principales

| Variable | Valeur locale par defaut | Description |
|---|---|---|
| `APP_ENV` | `development` | Environnement applicatif |
| `POSTGRES_DB` | `nutrimatch` | Nom de la base PostgreSQL |
| `POSTGRES_USER` | `postgres` | Compte admin PostgreSQL local |
| `POSTGRES_PASSWORD` | `postgres` | Mot de passe admin local |
| `APP_DB_USER` | `app` | Compte applicatif utilise par l'API |
| `APP_DB_PASSWORD` | `password123` | Mot de passe du compte applicatif |
| `POSTGRES_PORT` | `5432` | Port PostgreSQL expose sur la machine |
| `API_PORT` | `8080` | Port de l'API |
| `FRONTEND_PORT` | `3000` | Port du frontend |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | URL publique de l'API pour le frontend |
| `FRONTEND_BASE_URL` | `http://localhost:3000` | URL publique du frontend |
| `CORS_ORIGINS` | `http://localhost:3000` | Origines autorisees par CORS |
| `TRUSTED_ORIGINS` | `http://localhost:3000` | Origines autorisees pour CSRF |
| `REDIS_URL` | `redis://redis:6379/0` | Connexion Redis interne Docker |

### Secrets applicatifs

Les valeurs de `.env.docker.example` permettent une demonstration locale. Pour un usage serieux, generer des valeurs distinctes.

| Variable | Regle |
|---|---|
| `JWT_SECRET` | 32 caracteres minimum, idealement 64+ aleatoires |
| `REFRESH_TOKEN_PEPPER` | 32 caracteres minimum, distinct de `JWT_SECRET` |
| `HEALTH_DATA_ENCRYPTION_KEY` | Exactement 32 caracteres pour AES-256-GCM |
| `MFA_SECRET_ENCRYPTION_KEY` | Exactement 32 caracteres, distincte de la cle sante |
| `SENSITIVE_DATA_INDEX_KEY` | 32 caracteres minimum, distincte des autres cles |

PowerShell pour generer un secret long:

```powershell
[Convert]::ToHexString([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

PowerShell pour generer une cle de 32 caracteres:

```powershell
[Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(24))
```

Linux/macOS:

```bash
openssl rand -hex 32
openssl rand -base64 24
```

### IA Google optionnelle

| Variable | Description |
|---|---|
| `GOOGLE_AI_API_KEY` | Cle Google AI Studio. Peut rester vide. |
| `GOOGLE_AI_BASE_URL` | URL Google Generative Language API |
| `GOOGLE_AI_MODEL` | Modele utilise, par defaut `gemini-2.5-flash` |

Si `GOOGLE_AI_API_KEY` est vide, invalide ou limitee par quota, l'application affiche les recettes backend-safe sans inventer d'explication locale.

### OIDC optionnel

OIDC reste inactif tant que les variables suivantes restent vides:

```env
OIDC_ISSUER_URL=
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=
```

Si OIDC est active, les trois variables doivent etre configurees ensemble.

### WebAuthn local

```env
WEBAUTHN_RP_ID=localhost
WEBAUTHN_ORIGINS=http://localhost:3000
```

En production, ces valeurs doivent correspondre au domaine HTTPS final.

## Base de donnees

### Connexion depuis un client SQL

| Champ | Valeur locale |
|---|---|
| Driver | PostgreSQL |
| Host | `localhost` |
| Port | `5432` |
| User | `postgres` |
| Password | `postgres` |
| Database | `nutrimatch` |
| URL JDBC | `jdbc:postgresql://localhost:5432/nutrimatch` |

Schemas a afficher:

```text
catalog
health
identity
recommendation
security
public
```

### Verification avec psql

```powershell
docker exec -it nutrimatch-db-1 psql -U postgres -d nutrimatch
```

Commandes utiles:

```sql
\dn
\dt catalog.*
\dt health.*
\dt identity.*
\dt recommendation.*
\dt security.*
SELECT extname FROM pg_extension WHERE extname = 'vector';
```

### Reset complet de developpement

```powershell
docker compose down -v
docker compose up --build -d
```

`down -v` supprime le volume PostgreSQL et remet la base a zero.

## Flux de test manuel

1. Lancer `docker compose up --build -d`.
2. Ouvrir `http://localhost:3000`.
3. Creer un compte via `/signup` ou `/register`.
4. Se connecter via `/signin` ou `/login`.
5. Completer le profil dans l'onboarding.
6. Verifier l'affichage des recommandations.
7. Verifier le compte a rebours de 24 heures.
8. Tester le bouton de retry IA si l'IA est indisponible.
9. Choisir une recette.
10. Verifier que la recette choisie est affichee seule et mise en cache.
11. Tester `/signout`.
12. Verifier qu'une page protegee redirige vers `/login` sans session.

## Developpement hors Docker

Le mode hors Docker est optionnel. Il necessite PostgreSQL avec pgvector et Redis installes localement.

### Backend seul

```powershell
cd backend
Copy-Item .env.example .env
go run ./cmd/api
```

Migrations locales:

```powershell
cd backend
goose -dir .\migrations\ postgres "postgres://postgres:postgres@localhost:5432/nutrimatch?sslmode=disable" up
```

### Frontend seul

```powershell
cd frontend
Copy-Item .env.example .env.local
npm install
npm run dev
```

Le frontend attend l'API sur:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Tests et verification

### Backend

```powershell
cd backend
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

### Frontend

```powershell
cd frontend
npm run lint
npm run build
npm audit --omit=dev
```

### Docker

```powershell
docker compose up --build -d
Invoke-RestMethod http://localhost:8080/api/v1/health
docker compose ps
```

## Depannage

### Port deja utilise

Ports par defaut:

| Service | Port |
|---|---:|
| Frontend | `3000` |
| API | `8080` |
| PostgreSQL | `5432` |

Windows:

```powershell
Get-NetTCPConnection -LocalPort 3000,8080,5432 -ErrorAction SilentlyContinue | Select-Object LocalPort,OwningProcess
Get-Process -Id <PID>
Stop-Process -Id <PID>
```

Linux/macOS:

```bash
lsof -i :3000
lsof -i :8080
lsof -i :5432
```

Changer temporairement les ports:

```powershell
$env:FRONTEND_PORT="13000"
$env:API_PORT="18080"
$env:POSTGRES_PORT="15433"
$env:NEXT_PUBLIC_API_URL="http://localhost:18080"
$env:FRONTEND_BASE_URL="http://localhost:13000"
$env:CORS_ORIGINS="http://localhost:13000"
$env:TRUSTED_ORIGINS="http://localhost:13000"
$env:WEBAUTHN_ORIGINS="http://localhost:13000"
docker compose up --build -d
```

### `extension "vector" is not available`

Le PostgreSQL utilise ne contient pas pgvector. Utiliser la base fournie par Docker Compose.

### `health data encryption key must be 32 bytes`

`HEALTH_DATA_ENCRYPTION_KEY` doit faire exactement 32 caracteres.

### `OIDC_CLIENT_ID is required when OIDC is enabled`

Une variable OIDC est configuree sans les autres. Soit configurer OIDC completement, soit laisser toutes les variables OIDC vides.

### `ERR_EMPTY_RESPONSE`

Verifier que les conteneurs sont demarres:

```powershell
docker compose ps
docker compose logs --tail=100 api
docker compose logs --tail=100 frontend
```

### Quota IA atteint

Si Google AI renvoie un quota ou rate limit, les recettes restent disponibles sans explication IA. Le backend retente les erreurs 429 avant de retourner l'erreur structuree.

## Securite

Mesures principales implementees:

- mots de passe hashes avec Argon2id;
- refresh tokens hashes et stockes cote serveur;
- refresh token transmis par cookie HttpOnly;
- access token conserve en memoire cote frontend;
- CSRF signe et lie a la session pour les mutations sensibles;
- CORS et origines de confiance configurables;
- rate limiting via Redis;
- une seule session active selon la logique applicative;
- MFA optionnel avec TOTP et Passkeys WebAuthn;
- donnees sante sensibles chiffrees au repos cote application;
- index sensibles haches pour limiter l'exposition;
- schemas PostgreSQL separes par domaine;
- audit trail append-only avec chainage de hash;
- retention configurable;
- logs sans secrets ni corps bruts de fournisseur IA;
- catalogue local valide avant toute intervention IA;
- hard constraints prioritaires sur IA, score et similarite;
- absence d'explications mock ou fallback presentees comme IA;
- headers de securite et CSP cote frontend.

Limites volontaires du mode local:

- HTTP local pour faciliter la demonstration;
- pas de TLS configure dans Docker local;
- pas de vault externe;
- pas de SIEM externe;
- cle IA optionnelle.

Pour une production reelle:

- utiliser HTTPS;
- mettre `COOKIE_SECURE=true`;
- remplacer tous les secrets;
- utiliser des origines HTTPS strictes;
- stocker les secrets dans un gestionnaire adapte;
- superviser les logs et evenements d'audit.

## Distribution du projet

Inclure les sources et les fichiers de configuration exemple:

```text
docker-compose.yml
.env.docker.example
README.md
docker/
backend/
frontend/
doc/
```

Exclure les fichiers locaux, caches et artefacts generes:

```text
.git/
.idea/
.gocache/
.env
backend/.env
frontend/.env.local
frontend/node_modules/
frontend/.next/
frontend/build/
```
