# NutriMatch Backend (Go)

Backend Go securise pour NutriMatch, base sur Gin + GORM + PostgreSQL, avec clean architecture, JWT, OIDC, CSRF, chiffrement applicatif cible sur les donnees de sante, audit trail append-only et orchestration hybride des recommandations.

## Prerequis
- Go 1.25+
- PostgreSQL
- Goose (migrations)

## Configuration
- Copiez `.env.example` vers `.env` et ajustez les valeurs.
- Les durees des tokens JWT sont gerees via `ACCESS_TOKEN_TTL_MINUTES` et `REFRESH_TOKEN_TTL_HOURS`.
- `REFRESH_TOKEN_PEPPER` sert au hachage des refresh tokens stockes en table `sessions`.
- `JWT_SECRET`, `REFRESH_TOKEN_PEPPER` et `HEALTH_DATA_ENCRYPTION_KEY` doivent etre distincts et faire au moins 32 caracteres.
- `COOKIE_PATH_REFRESH` permet de limiter la portee du cookie refresh aux endpoints d'auth.
- `COOKIE_PATH_CSRF`, `COOKIE_NAME_CSRF`, `CSRF_HEADER_NAME` et `CSRF_TTL_MINUTES` pilotent la protection CSRF double-submit signee.
- `TRUSTED_ORIGINS` definit les origines autorisees pour les endpoints sensibles bases sur cookie (`csrf`, `register`, `login`, `refresh`, `logout`).
- `FRONTEND_BASE_URL` et `OIDC_FRONTEND_SUCCESS_URL` doivent pointer vers le frontend qui consomme le callback OIDC.
- `SPOONACULAR_API_KEY` et `SPOONACULAR_BASE_URL` pour l'API recettes.
- `GOOGLE_AI_API_KEY`, `GOOGLE_AI_BASE_URL`, `GOOGLE_AI_MODEL` pour Google AI Studio. Le modele stable par defaut est `gemini-2.5-flash`.
- `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL`, `OIDC_SCOPES`, `OIDC_PROVIDER_NAME` activent OpenID Connect.

## Architecture securite et donnees
- Schemas PostgreSQL separes:
  - `identity.*` pour les comptes, sessions et identites federes
  - `health.*` pour les profils, contraintes, profils nutritionnels et traces de recommandations
  - `security.*` pour l'audit trail append-only
- Les contraintes sante sensibles (ex. medicaments) sont chiffrees cote application avant persistance.
- Les recommandations suivent une hierarchie explicite:
  - regles deterministes et firewall nutritionnel
  - filtres sante et contraintes derivees
  - API recettes externe
  - reranking IA uniquement apres validation et desactive sur profils sensibles

## Migrations (Goose)
- Les tables sont creees via une seule migration initiale.

Commandes:
```powershell
$env:GOOSE_DRIVER="postgres";
$env:GOOSE_DBSTRING="postgres://user:password@localhost:5432/nutrimatch?sslmode=disable";
goose -dir .\migrations up
```

## Lancer l'API
```powershell
go run .\cmd\api
```

## Tests
```powershell
go test .\...
```

## Endpoints
- `GET /api/v1/health`
- `GET /api/v1/auth/csrf`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/oidc/login`
- `GET /api/v1/auth/oidc/callback`
- `POST /api/v1/profile` (auth requise)
- `GET /api/v1/profile` (auth requise)
- `GET /api/v1/profile/nutrition` (auth requise)
- `GET /api/v1/recommendations/:profileId` (auth requise)
- `GET /api/v1/recommendations/:profileId/trace` (auth requise)
- `GET /api/v1/recommendations/:profileId/explanation?mealId=...` (auth requise)

Notes securite:
- Les mots de passe font au minimum 12 caracteres.
- En production, TLS est attendu pour la base de donnees, les cookies doivent etre `Secure`, et les origines doivent etre en `https`.
- Les traces d'appels externes et les decisions de filtrage sont stockees pour auditabilite.
- Les sorties IA ne sont jamais source d'autorite: elles ne peuvent qu'ajuster le classement final apres passage du firewall deterministe.

Compatibilite frontend:
- `POST /api/v1/profile`
- `GET /api/v1/profile`
- `GET /api/v1/profile/nutrition`
- `GET /api/v1/recommendations/:profileId`
