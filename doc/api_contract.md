# API Contract NutriMatch

Contrat courant de l'API `v1`.

## Envelope

Toutes les reponses JSON de succes utilisent l'enveloppe suivante :

```json
{
  "data": {},
  "meta": {
    "requestId": "uuid",
    "timestamp": "2026-04-21T12:00:00Z"
  }
}
```

Toutes les erreurs JSON utilisent l'enveloppe suivante :

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "validation failed"
  },
  "meta": {
    "requestId": "uuid",
    "timestamp": "2026-04-21T12:00:00Z"
  }
}
```

## Endpoints proteges

- `GET /api/v1/auth/whoami`
- `POST /api/v1/profile`
- `GET /api/v1/profile`
- `GET /api/v1/profile/nutrition`
- `GET /api/v1/profile/ingredients/suggest`
- `GET /api/v1/recommendations/:profileId`
- `GET /api/v1/recommendations/:profileId/trace`
- `GET /api/v1/recommendations/:profileId/explanation?mealId=...`

## Notes de contrat

- `GET /api/v1/profile` masque `constraints.medications` par defaut.
- Pour demander explicitement les medicaments dechiffres, utiliser `GET /api/v1/profile?includeSensitive=true`.
- `GET /api/v1/auth/whoami` retourne l'etat de session courant, la methode d'authentification et `hasProfile/profileId`.
- `204 No Content` reste utilise uniquement pour `POST /api/v1/auth/logout`.
