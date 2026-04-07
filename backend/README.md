# NutriMatch Backend (Go)

Backend Go securise pour NutriMatch, base sur Gin + GORM + PostgreSQL, avec clean architecture, JWT et Argon2id.

## Prerequis
- Go 1.25+
- PostgreSQL
- Goose (migrations)

## Configuration
- Copiez `.env.example` vers `.env` et ajustez les valeurs.
- Les durees des tokens JWT sont gerees via `ACCESS_TOKEN_TTL_MINUTES` et `REFRESH_TOKEN_TTL_HOURS`.
- `REFRESH_TOKEN_PEPPER` sert au hachage des refresh tokens stockes en table `sessions`.
- `SPOONACULAR_API_KEY` et `SPOONACULAR_BASE_URL` pour l'API recettes.
- `GOOGLE_AI_API_KEY`, `GOOGLE_AI_BASE_URL`, `GOOGLE_AI_MODEL` pour Google AI Studio.

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
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/profile` (auth requise)
- `GET /api/v1/profile` (auth requise)
- `GET /api/v1/recommendations/:profileId` (auth requise, stub)

Compatibilite frontend:
- `POST /api/profile`
- `GET /api/recommendations/:profileId`
