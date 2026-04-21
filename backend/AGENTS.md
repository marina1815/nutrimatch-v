# Backend AGENTS

## Objectif

Le backend NutriMatch est l'autorite metier et securite du projet.

Il doit:

- valider et normaliser les saisies utilisateur
- calculer le profil nutritionnel derive
- appliquer les filtres sante et nutritionnels deterministes
- orchestrer la recherche hybride
- tracer les decisions et appels externes

## Architecture a respecter

- `cmd/api/` point d'entree
- `internal/http/handlers/` adaptation HTTP uniquement
- `internal/http/middleware/` controles transverses
- `internal/services/` logique metier
- `internal/repository/` interfaces
- `internal/repository/gorm/` persistance concrete
- `internal/security/` primitives de securite
- `internal/clients/` integrations externes

Ne pas deplacer de logique sensible dans les handlers.

## Regles de securite

- Toute donnee sante sensible doit etre minimisee, chiffree si necessaire, et jamais exposee sans besoin explicite
- L'IA est non autoritative: elle ne doit jamais contourner les filtres deterministes
- Les appels sortants doivent etre traces sans fuite de secrets
- Toute nouvelle route protegee doit passer par les middlewares d'auth appropries
- Les erreurs doivent suivre le contrat JSON uniforme `error/meta`

## Contrat API

- Les reponses de succes suivent `data/meta`
- Les erreurs suivent `error/meta`
- `logout` reste en `204 No Content`
- `whoami` est la route de verite pour l'etat de session courant

## Base de donnees

- La base suit une creation propre sans `ALTER` intermediaire
- Respecter la separation logique `identity`, `catalog`, `health`, `recommendation`, `security`
- Toute nouvelle persistance doit conserver l'integrite transactionnelle

## Tests attendus

Avant de considerer un changement backend comme termine:

- `go test ./...` doit passer
- les cas de securite et de degradation gracieuse doivent etre couverts si le changement touche auth, profil ou recommandations
