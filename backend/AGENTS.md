# AGENTS.md

## Portee
- Le code actif est dans `frontend/` (Next.js App Router). Le `backend/` est present mais sans implementation visible.
- Conventions agent Next.js: voir `frontend/AGENTS.md` (version Next.js non standard, lire les guides dans `node_modules/next/dist/docs/`).

## Architecture et flux
- Pages App Router dans `frontend/src/app/`: landing `page.tsx`, auth `app/(auth)/{login,register}/page.tsx`, onboarding `app/onboarding/page.tsx`, profil `app/profile/page.tsx`, resultats `app/results/page.tsx`.
- Flux principal: onboarding -> resultats. `app/onboarding/page.tsx` utilise `useProfileForm` et redirige vers `/results` apres l'etape 4.
- Donnees profil persistees en localStorage via `frontend/src/hooks/useProfileForm.ts` (cle `nutrimatch-profile`).
- Resume profil + indicateurs sante calcules dans `frontend/src/lib/health.ts` et affiches dans `app/profile/page.tsx`.
- Les resultats actuels sont simules via `frontend/src/lib/mock-data.ts` et rendus par `components/results/RecommendationList.tsx` + `MealCard.tsx`.

## Conventions UI
- Classes CSS globales prefixees `nm-` definies dans `frontend/src/app/globals.css` et reutilisees par les composants UI.
- Certaines pages utilisent des styles inline via `<style>{`...`}</style>` (ex: `app/page.tsx`, `app/(auth)/login/page.tsx`, `app/(auth)/register/page.tsx`).
- Composants UI simples dans `frontend/src/components/ui/` (ex: `Button`, `Card`, `Input`, `Select`, `Checkbox`).
- Types metier centralises dans `frontend/src/lib/types.ts`; options de select et labels dans `frontend/src/lib/constants.ts`.

## Validation et formulaires
- Validation par etape dans `frontend/src/lib/validation.ts`; `useProfileForm.next()` bloque la progression si erreurs.
- Les formulaires d'onboarding sont decoupes en etapes: `components/forms/{PersonalInfoStep,LifestyleStep,PreferencesStep,ConstraintsStep}.tsx`.

## Integrations et API
- Appel API via `frontend/src/lib/api.ts` avec `NEXT_PUBLIC_API_URL` (defaut `http://localhost:8080`).
- Endpoints attendus: `POST /api/profile` et `GET /api/recommendations/:profileId`.

## Dev workflows (frontend)
- Scripts npm dans `frontend/package.json`: `dev`, `build`, `start`, `lint`.
- Alias TypeScript `@/*` -> `frontend/src/*` configure dans `frontend/tsconfig.json`.

