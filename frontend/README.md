# NutriMatch Frontend (Next.js)

Frontend Next.js de NutriMatch, connecte au backend Go reel via l'API REST `v1`.

## Objectif

Le frontend couvre le parcours principal:

- authentification locale ou OIDC
- onboarding nutritionnel multi-etapes
- consultation du profil persistant
- affichage des recommandations et de leurs explications

Le design visuel existant est preserve autant que possible. La logique sensible reste cote backend.

## Prerequis

- Node.js 20+
- Backend NutriMatch accessible via `NEXT_PUBLIC_API_URL`

## Configuration

Copiez vos variables d'environnement frontend si necessaire, puis configurez:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Scripts

```powershell
npm run dev
npm run lint
npx tsc --noEmit
npm run build
```

Le build de production utilise `next build --webpack` pour rester fiable sur l'environnement Windows actuel du projet. La sortie est ecrite dans `frontend/build/`.

## Architecture

- `src/app/`
  - `(auth)/login`, `(auth)/register`
  - `auth/oidc/callback`
  - `onboarding`
  - `profile`
  - `results`
- `src/components/`
  - `forms/` pour les etapes du profil
  - `results/` pour les cartes et explications
  - `ui/` pour les briques de presentation
- `src/lib/`
  - `api.ts` client API uniforme
  - `session.ts` stockage navigateur borne a `sessionStorage`
  - `validation.ts` validation formulaire
  - `profile-normalization.ts` sanitisation des saisies

## Contrat avec le backend

- Toutes les reponses JSON backend sont enveloppees dans `data/meta`
- Toutes les erreurs JSON backend sont enveloppees dans `error/meta`
- `POST /api/v1/auth/logout` reste volontairement en `204 No Content`
- `GET /api/v1/profile` masque `constraints.medications` par defaut
- `GET /api/v1/auth/whoami` permet de decider si l'utilisateur part vers `/onboarding` ou `/results`

## Hygiene securite

- Les donnees sensibles ne sont plus stockees en `localStorage`
- Le jeton d'acces et le brouillon de profil restent limites a `sessionStorage`
- Les messages backend bruts ne sont pas affiches directement a l'utilisateur
- Les flux cookie sensibles utilisent CSRF cote backend

## Etat de verification

- `npm run lint` passe
- `npx tsc --noEmit` passe
- `npm run build` passe avec la configuration actuelle
