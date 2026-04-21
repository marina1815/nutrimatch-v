# Security Controls Matrix

Matrice de travail pour aligner l'implementation NutriMatch avec `doc/stride.md`, le DFD et les controles applicatifs reellement en place.

## Frontieres de confiance

- `client -> api`: navigateur vers API REST Go
- `api -> persistence`: services backend vers PostgreSQL
- `api -> external_recipe_api`: backend vers Spoonacular
- `api -> ai_rerank`: backend vers Google AI Studio

## Menaces, controles, preuves

| Menace | Controle actuel | Preuve dans le projet | Statut |
| --- | --- | --- | --- |
| Spoofing sur flux cookie auth | CSRF signe, origine verifiee, cookie refresh borne au path auth | `backend/internal/security/csrf.go`, `backend/internal/http/routes/router.go`, `backend/internal/http/handlers/auth_handler.go` | partiel |
| Broken authentication | access token JWT signe, refresh token hache + rotation, blocage temporaire login, rate limiting auth | `backend/internal/security/tokens.go`, `backend/internal/services/auth_service.go`, `backend/internal/http/middleware/ratelimit.go` | en place |
| BOLA / IDOR sur profil et recommandations | verification explicite `userID/profileID`, route `whoami`, access policy service | `backend/internal/http/middleware/auth.go`, `backend/internal/services/access_policy_service.go`, `backend/internal/services/recommendation_service.go` | en place |
| Tampering sur payload profil | JSON strict, champs inconnus refuses, validation de coherence et de cardinalite | `backend/internal/http/handlers/binding.go`, `backend/internal/http/handlers/profile_handler.go`, `frontend/src/lib/validation.ts` | en place |
| Information disclosure des donnees sante | chiffrement applicatif cible, medicaments masques par defaut, pseudonymisation de traces IP/User-Agent | `backend/internal/security/cipher.go`, `backend/internal/http/handlers/profile_handler.go`, `backend/internal/http/handlers/security_helpers.go` | en place |
| Recommendation poisoning / fail-safe sante | firewall nutritionnel deterministe, contraintes dures, raisons de rejet/acceptation tracees | `backend/internal/services/recommendation_service.go`, `backend/internal/http/routes/router_test.go` | en place |
| Unsafe AI output | IA non autoritative, prompt minimise, validation stricte des IDs/bonus, bypass sur profils sensibles | `backend/internal/services/recommendation_service.go` | en place |
| Repudiation | audit trail applicatif, traces de recommandation, corrrelation requestId/runId | `backend/internal/services/audit_service.go`, `backend/internal/models/audit_event.go`, `doc/api_contract.md` | partiel |
| Upstream outage / resilience | cache L1/L2, circuit breaker simple, fallback sans IA, degrades propres | `backend/internal/clients/spoonacular`, `backend/internal/services/recommendation_service.go`, tests routes/services | en place |
| Secrets leakage | redaction/pseudonymisation des traces, config stricte au demarrage | `backend/internal/config/config.go`, `backend/internal/http/handlers/security_helpers.go` | partiel |
| Denial of service par payloads complexes | body limit, budget texte/signaux, quotas recommandations, rate limiting | `backend/internal/config/config.go`, `backend/internal/http/handlers/profile_handler.go`, `backend/internal/security/quota.go` | en place |
| SSRF future si URLs externes acceptees | aucune URL externe utilisateur acceptee aujourd'hui | architecture actuelle | hors scope actuel |

## Alignement normes

### OWASP ASVS 5.0

- Couverture solide:
  - validation d'entree
  - gestion de session
  - controle d'acces objet
  - journalisation metier
  - protection CSRF
  - minimisation d'exposition
- Reste a completer:
  - audit trail immuable prouve au niveau infra/DB
  - rotation/gestion operationnelle des secrets
  - verification email / reset / MFA si retenus
  - preuve formelle des frontieres de confiance

### OWASP API Security Top 10 2023

- Bien couvert:
  - `API1 Broken Object Level Authorization`
  - `API2 Broken Authentication`
  - `API3 Broken Object Property Level Authorization`
  - `API4 Unrestricted Resource Consumption`
  - `API10 Unsafe Consumption of APIs` partiellement via circuit breaker, cache, validation et fallback
- A renforcer:
  - `API8 Security Misconfiguration` en production
  - inventaire/versioning final complet des endpoints

### NIST SP 800-63B-4

- Base correcte pour les sessions et secrets applicatifs
- Reste a evaluer si le produit vise un niveau plus eleve:
  - MFA ou passkeys
  - preuve operationnelle OIDC contre un vrai fournisseur
  - parcours verification email / recuperation de compte

## Limites connues

- L'audit trail est append-only au niveau applicatif, mais l'immutabilite forte depend encore de la base et des roles d'exploitation.
- La segregation identite/sante est logique et schema-based; une isolation de privileges encore plus stricte reste a faire en production.
- Le build frontend a ete stabilise en `webpack` sur l'environnement Windows actuel; Turbopack reste sensible aux permissions filesystem locales.
- Les profils similaires / couche vectorielle sont conceptualises et relies au modele de donnees, mais l'expansion semantique reste volontairement limitee.

## Preuves de verification

- `go test ./...`
- `npm run lint`
- `npx tsc --noEmit`
- `npm run build`
