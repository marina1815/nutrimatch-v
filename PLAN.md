# PLAN.md

Plan directeur de refonte et de finalisation de `NutriMatch`.

Objectif global : livrer un frontend et un backend complets, relies entre eux, coherents avec Spoonacular, durcis cote securite, avec une base SQL propre, une couche vectorielle correctement conceptualisee, une logique metier robuste, et une integration IA strictement validee.

Convention :
- `[x]` termine
- `[~]` en cours
- `[ ]` a faire

## 0. Etat actuel

- [x] Audit initial du depot effectue
- [x] Lecture des documents projet (`AGENTS.md`, `README.md`, `doc/nutrimatch.md`, `doc/stride.md`, `doc/dfd_v3.md`)
- [x] Lecture et analyse du schema [dfd_v3.png](</D:/Coding/NutriMatch/doc/dfd_v3.png>)
- [x] Revue architecture frontend existante
- [x] Revue architecture backend existante
- [x] Identification des principaux ecarts frontend/backend/securite
- [x] Verification backend : `go test ./...` passe
- [x] Verification frontend initiale : `npm run lint` a revele des dettes techniques reelles
- [x] Verification frontend apres corrections : `npm run lint` passe
- [x] Creation du present plan de travail

## 1. Gouvernance de la refonte

- [~] Geler le contrat cible entre frontend et backend avant les gros changements
- [~] Definir un vocabulaire metier canonique unique pour ingredients, allergies/intolerances, maladies/conditions, objectifs nutritionnels, types de repas et preferences
- [ ] Definir les frontieres de confiance reelles du systeme
- [ ] Transformer `doc/stride.md` en exigences de securite verifiables dans le code
- [x] Definir une strategie de migration "clean rewrite" sans `ALTER`, avec schema initial propre

## 2. Documentation Spoonacular et compatibilite formulaire

- [x] Recenser precisement les capacites Spoonacular utiles au projet
- [~] Valider et documenter les endpoints Spoonacular a utiliser
- [x] Lister les champs Spoonacular supportes par le moteur de recherche
- [~] Definir un mapping strict formulaire -> parametres Spoonacular
- [~] Definir une normalisation FR -> valeurs canoniques internes -> valeurs Spoonacular
- [~] Prevoir le traitement des cas non mappables a Spoonacular
- [~] Definir la strategie pour les ingredients libres saisis par l'utilisateur
- [ ] Definir les limites du formulaire pour rester compatible avec l'API sans perdre l'expressivite metier

## 3. Modele de donnees cible

### 3.1 Base SQL

- [x] Redessiner le schema SQL complet depuis zero
- [x] Separer proprement les domaines : `identity`, `catalog`, `health`, `recommendation`, `security`
- [x] Prevoir des tables de reference pour ingredients, alias, intolerances, conditions, meal styles, cuisines, diets et mappings Spoonacular
- [x] Prevoir un schema de persistance du profil utilisateur complet
- [x] Prevoir un schema de persistance de la trace de decision
- [x] Prevoir un schema de persistance des appels externes et erreurs
- [x] Prevoir des index adaptes aux usages principaux
- [x] Definir contraintes d'integrite, unicite, FK, `CHECK`, valeurs bornees
- [ ] Definir politique de retention pour audit, sessions, cache et traces

### 3.2 Couche vectorielle

- [x] Concevoir la couche vectorielle avant implementation
- [x] Definir son role reel
- [x] Definir les embeddings a stocker
- [x] Definir les metadonnees a stocker avec chaque vecteur
- [x] Choisir la strategie d'implementation : PostgreSQL + `pgvector`
- [x] Definir les politiques de recalcul, invalidation et versionnement
- [x] Definir les garde-fous securite / confidentialite de la couche vectorielle

## 4. Backend : refonte fonctionnelle et securite

### 4.1 Contrat API

- [~] Redefinir le contrat API frontend/backend complet
- [~] Versionner clairement les DTO d'entree/sortie
- [x] Uniformiser les reponses d'erreur
- [~] Uniformiser les codes HTTP
- [x] Uniformiser les identifiants, timestamps et metadonnees de tracage

### 4.2 Authentification et session

- [~] Finaliser le flux register/login/logout/refresh end-to-end
- [~] Definir la strategie finale access token / refresh token
- [~] Reduire la surface d'attaque des sessions cote navigateur
- [x] Renforcer rotation, revocation, expiration absolue et idle timeout
- [ ] Ajouter gestion multi-session explicite
- [ ] Ajouter endpoint "whoami / session courante" si utile
- [ ] Ajouter flux mot de passe oublie / reset si retenu
- [ ] Ajouter verification email si retenu
- [ ] Evaluer MFA / passkeys comme extension eventuelle

### 4.3 Profil utilisateur

- [x] Refaire la persistance du profil sur le nouveau schema
- [~] Introduire normalisation et validation forte cote backend
- [~] Refuser toute donnee incoherente ou ambigue
- [x] Chiffrer correctement les donnees de sante sensibles au repos
- [~] Definir la politique d'exposition minimale des donnees de profil
- [x] Ajouter endpoint de lecture/edition coherent avec le frontend

### 4.4 Taxonomie metier

- [~] Implementer un registre metier unifie
- [~] Implementer la normalisation multilingue et synonymique
- [ ] Definir les regles de priorite entre allergie, exclusion manuelle, preference positive et recommandation IA

### 4.5 Orchestration Spoonacular

- [x] Reecrire le client Spoonacular sur un contrat clair
- [x] Gerer les parametres de recherche avancee
- [~] Ajouter recuperation complete nutrition + ingredients + metadata
- [x] Ajouter timeouts, retries limites, circuit breaker simple si utile
- [~] Ajouter cache des reponses externes branche au client
- [x] Gerer proprement quota, erreurs 4xx/5xx, timeouts et reponses partielles
- [~] Journaliser les appels sortants sans exposer de secrets

### 4.6 Moteur de recommandation

- [~] Redefinir la pipeline de recommandation
- [~] Separer filtrage de securite dur, filtrage metier, enrichissement Spoonacular, scoring deterministe, expansion vectorielle et reranking IA valide
- [x] Faire des regles medicales des contraintes dures la ou necessaire
- [x] Implementer les plafonds / seuils nutritionnels complets
- [x] Gerer explicitement `required_tags`, nutriments min/max, sodium, sucre, proteines, etc.
- [x] Definir des raisons d'acceptation / rejet auditables
- [x] Prevoir un mode "aucun resultat sur" propre et explicable

### 4.7 IA et validation

- [~] Limiter l'IA a un role non autoritatif
- [ ] Definir precisement ce qui peut etre envoye au modele
- [~] Minimiser/anonymiser les donnees avant appel IA
- [~] Empecher que l'IA contourne les regles sante
- [~] Valider strictement la sortie IA avant usage
- [ ] Prevoir fallback sans IA
- [~] Tracer la contribution IA au score final

### 4.8 Controles de securite backend

- [~] Renforcer la politique de controle d'acces
- [~] Verifier explicitement tous les risques d'IDOR/BOLA
- [~] Revoir CSRF/cookies/CORS selon le flux final reel
- [~] Ajouter protection anti-bruteforce et anti-enumeration
- [~] Durcir rate limiting et quotas pour environnement multi-instance
- [~] Ajouter validation de taille, cardinalite et complexite des payloads
- [~] Ajouter politique de logs de securite sans fuite de donnees sensibles
- [ ] Ajouter protection contre SSRF si des URLs externes sont un jour acceptees
- [~] Ajouter politique de secrets et rotation
- [~] Ajouter strategie de hardening production

## 5. Frontend : integration sans casser le design

### 5.1 Principes

- [x] Ne pas modifier le design visuel hors necessite technique/securite
- [x] Supprimer les flows factices et brancher le vrai backend
- [~] Reduire au maximum l'exposition des donnees sensibles cote navigateur

### 5.2 Auth frontend

- [~] Connecter register/login/logout/refresh au backend reel
- [x] Ajouter recuperation prealable du token CSRF si flux cookie maintenu
- [x] Gerer `credentials: "include"` si requis
- [x] Gerer le Bearer token si l'API le requiert
- [~] Ajouter gestion propre des erreurs et expirations de session

### 5.3 Formulaire profil

- [~] Refaire le formulaire pour qu'il utilise le vocabulaire metier canonique
- [~] Ajouter autocompletion / suggestion d'ingredients compatible Spoonacular
- [~] Ajouter validation frontend alignee sur la validation backend
- [x] Corriger les incoherences de types existantes
- [x] Corriger les champs mal nommes (`mealTypes` vs `mealStyles`, etc.)
- [x] Brancher l'envoi du profil au backend reel
- [x] Recuperer et afficher le profil persistant depuis le backend

### 5.4 Reduction de surface d'attaque frontend

- [x] Retirer le stockage local persistant de donnees de sante sensibles
- [x] Definir ce qui peut eventuellement rester en `sessionStorage`
- [~] Supprimer les donnees sensibles du DOM, des logs et des erreurs affichees
- [~] Eviter toute injection cote rendu
- [x] Reduire les usages de `any`
- [x] Corriger les erreurs ESLint et les patterns React a risque
- [ ] Verifier les dependances et la configuration Next.js

### 5.5 Resultats et explications

- [x] Connecter la page resultats aux vraies recommandations backend
- [x] Afficher les raisons de match de facon fiable
- [x] Ajouter affichage des explications / traces si utile
- [x] Gerer les cas "aucun resultat sur"

## 6. Securite, normes et conformite projet

- [ ] Aligner le projet autant que possible sur OWASP ASVS 5.0.0
- [ ] Aligner l'API sur OWASP API Security Top 10 2023
- [ ] Aligner l'authentification/sessions sur NIST SP 800-63B-4
- [ ] Utiliser les OWASP Cheat Sheets pertinentes
- [ ] Definir une matrice "menace -> controle -> preuve"
- [ ] Reviser `doc/stride.md` apres implementation reelle

## 7. Tests et verification

### 7.1 Backend

- [x] Ajouter tests unitaires sur la normalisation metier
- [ ] Ajouter tests unitaires sur les regles medicales
- [ ] Ajouter tests unitaires sur le moteur de scoring
- [x] Ajouter tests d'integration API auth
- [x] Ajouter tests d'integration API profil
- [x] Ajouter tests d'integration API recommandation
- [~] Ajouter tests de securite (BOLA/IDOR, CSRF, session rotation, refus des champs inconnus, quotas/rate limits)

### 7.2 Frontend

- [x] Corriger `npm run lint`
- [x] Faire passer `npm run build`
- [ ] Ajouter tests sur les flux critiques si stack retenue
- [x] Verifier l'absence de dependance aux mocks en production

### 7.3 End-to-end

- [~] Tester register -> login -> onboarding -> profil -> recommandations -> explication
- [x] Tester profils avec allergies
- [x] Tester profils avec maladies chroniques
- [x] Tester profils avec medicaments
- [x] Tester cas sans resultats surs
- [x] Tester degradation gracieuse si Spoonacular indisponible
- [x] Tester degradation gracieuse si IA indisponible

## 8. Nettoyage et documentation finale

- [~] Reecrire les README selon l'architecture finale
- [ ] Documenter les variables d'environnement reelles
- [x] Documenter le schema SQL final cible
- [x] Documenter la couche vectorielle finale cible
- [x] Documenter le contrat API final
- [x] Documenter les choix de securite et leurs limites
- [x] Documenter la logique metier finale
- [x] Supprimer le code mort, les mocks non necessaires et les anciens schemas

## 9. Ordre d'execution recommande

- [~] Phase 1 : figer vocabulaire metier + compatibilite Spoonacular
- [x] Phase 2 : redessiner schema SQL + vectoriel + migrations initiales
- [~] Phase 3 : refondre backend auth/profil/taxonomies/recommandation
- [~] Phase 4 : connecter et durcir le frontend sans toucher au design
- [ ] Phase 5 : integrer l'IA avec validation stricte
- [ ] Phase 6 : completer tests, docs, STRIDE final et nettoyage

## 10. Criteres de fin

- [x] Le frontend n'utilise plus de resultats mockes
- [x] Le frontend ne stocke plus en clair des donnees sante persistantes non necessaires
- [ ] Le formulaire est aligne sur la taxonomie metier et compatible Spoonacular
- [~] Le formulaire est aligne sur la taxonomie metier et compatible Spoonacular
- [ ] Le backend applique les regles medicales et nutritionnelles de facon deterministe
- [ ] L'IA ne peut pas contourner les regles de securite
- [x] La base SQL est propre, coherente et recreable depuis zero
- [x] La couche vectorielle est correctement definie et reliee a un besoin metier reel
- [x] `go test ./...` passe
- [x] `npm run lint` passe
- [x] `npm run build` passe
- [ ] Le flux complet utilisateur fonctionne de bout en bout

## 11. References de securite et API a suivre

- [x] OWASP ASVS : https://owasp.org/www-project-application-security-verification-standard/
- [x] OWASP API Security Top 10 : https://owasp.org/API-Security/
- [x] OWASP Session Management Cheat Sheet : https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- [x] NIST SP 800-63B-4 : https://csrc.nist.gov/pubs/sp/800/63/b/4/final
- [~] Documentation Spoonacular utile au projet

## 12. Journal d'avancement

- [x] 2026-04-20 : audit complet initial, revue securite, revue docs, revue image DFD, validation backend, creation du plan directeur
- [x] 2026-04-20 : creation de `doc/spoonacular_contract.md`, ajout d'une taxonomie backend canonique avec tests, normalisation initiale allergies/conditions/styles de repas, alignement des types et options frontend, correction de `mealTypes` -> `mealStyles`, lint frontend au vert, backend toujours vert sur `go test ./...`
- [x] 2026-04-20 : suppression des flows frontend factices, ajout d'un client API avec CSRF + cookies + Bearer + refresh, soumission reelle du profil, chargement reel du profil et des recommandations, migration du brouillon profil de `localStorage` vers `sessionStorage`, suppression du mock de resultats, validation frontend via `lint` et `tsc --noEmit`
- [x] 2026-04-20 : formalisation de `doc/data_model_target.md` pour cadrer la refonte SQL et vectorielle, avec separation par domaines, tables de taxonomie, snapshots sante, audit, cache externe et strategie `pgvector`
- [x] 2026-04-20 : reecriture de `backend/migrations/0001_init.sql` avec schemas `identity/catalog/health/recommendation/security`, tables relationnelles de liaisons profil/taxonomie, caches externes, tables vectorielles `pgvector`, migration des regles medicales vers `catalog.medical_rules`, et durcissement des sessions
- [x] 2026-04-20 : refonte des modeles/repos backend pour persister preferences et contraintes via tables de liaison, ajout d'empreintes hachees IP/User-Agent, ajout d'un idle timeout de session, deplacement des runs/candidats de recommandation dans le schema `recommendation`, backend valide par `go test ./...`
- [x] 2026-04-21 : refonte du client Spoonacular avec options avancees bornees, erreurs upstream structurees et tests dedies ; renforcement du moteur de recommandation avec contraintes dures `required_tags`, plafonds medicaux explicites, enrichissement des tags derives et validation backend via `go test ./...`
- [x] 2026-04-21 : ajout de tests d'integration backend auth/profil/recommandation via `httptest`, couverture de rejets de securite (CSRF, origine non approuvee, bearer manquant, champs inconnus), et exposition frontend des traces/explications de recommandation
- [x] 2026-04-21 : ajout d'un cache persistant des reponses de recherche Spoonacular aligne sur le schema SQL cible, combine a un cache memoire L1, avec TTL configurable et tests dedies
- [x] 2026-04-21 : durcissement du reranking IA en mode non autoritatif avec prompt minimise, validation stricte des IDs et bonus retournes, conservation de l'explication deterministe et tracage explicite de la contribution IA
- [x] 2026-04-21 : extension des tests backend sur les acces inter-profils, les quotas de recommandation et le rate limiting auth ; assainissement des traces d'appels Spoonacular pour conserver une preuve technique sans stocker les details bruts de requete
- [x] 2026-04-21 : durcissement du login avec echec uniforme utilisateur inconnu / mot de passe invalide, verification crypto factice pour limiter l'enumeration par timing, suivi des echecs recentes par email/IP haches, blocage temporaire configurable et tests service/HTTP associes
- [x] 2026-04-21 : remplacement des quotas et du rate limiting purement memoire par un moteur partage de token bucket avec support persistant sur `security.rate_limit_buckets`, injection dans le middleware HTTP et les recommandations, et tests dedies du moteur de quota
- [x] 2026-04-21 : separation explicite de la pipeline interne de recommandation entre extraction des faits recette, hard filters, scoring deterministe et reranking IA optionnel, avec nouveaux tests cibles sur le filtrage dur et le scoring
- [x] 2026-04-21 : explicitation de l'etape `recipe_enrichment` issue de Spoonacular avec provenance des plans de recherche et du cache, plus separation de la similarite entre voisins deterministes et future expansion semantique/vectorielle, backend valide par `go test ./...`
- [x] 2026-04-21 : verification de coherence frontend apres les evolutions backend : `npm run lint` et `npx tsc --noEmit` restent verts ; `npm run build` reste bloque par un `EPERM` sur `.next` et non par une erreur applicative de compilation
- [x] 2026-04-21 : alignement supplementaire du frontend avec la taxonomie backend via sanitisation du profil, bornes de champs et validation plus stricte ; suppression de l'affichage des messages backend bruts sur login/register/onboarding/profile/results au profit de messages UI bornes, avec `npm run lint` et `npx tsc --noEmit` toujours verts
- [x] 2026-04-21 : ajout d'un circuit breaker simple au searcher Spoonacular avec seuil et cooldown configurables, ouverture seulement sur echecs upstream retryables, reutilisation prioritaire du cache, et validation backend via `go test ./...`
- [x] 2026-04-21 : extension du contrat profil avec `maxReadyTime`, `mealTypes`, `preferredCuisines` et `excludedCuisines`, propagation backend/frontend jusqu'a Spoonacular, ajout d'un endpoint protege de suggestion d'ingredients, validation canonique renforcee cote backend, autocompletion frontend sur les ingredients libres, et verification par `go test ./...`, `npm run lint` et `npx tsc --noEmit`
- [x] 2026-04-21 : alignement supplementaire sur le plan avec verification stricte de `HEALTH_DATA_ENCRYPTION_KEY` (exactement 32 caracteres), reduction de l'exposition par defaut des medicaments dans `GET /profile`, pseudonymisation des empreintes IP/User-Agent dans l'audit applicatif, et suppression du reliquat frontend `localStorage` pour le callback OIDC
- [x] 2026-04-21 : uniformisation du contrat JSON API avec enveloppes `data/meta` et `error/meta`, ajout d'un endpoint protege `auth/whoami`, alignement frontend sur le nouveau contrat et redirection login/OIDC selon l'existence d'un profil, valide par `go test ./...`, `npm run lint` et `npx tsc --noEmit`
- [x] 2026-04-21 : ajout de garde-fous backend/frontend contre les profils ambigus ou trop complexes (overlaps likes/dislikes, cuisines preferees/exclues, flags sante incoherents, budget texte et signaux libres bornes), plus tests backend sur le flux sensible complet `profil -> nutrition -> recommandations -> trace -> explication`, avec `go test ./...`, `npm run lint` et `npx tsc --noEmit` toujours verts
- [x] 2026-04-21 : couverture de robustesse du moteur de recommandation avec tests backend pour `no safe matches`, indisponibilite Spoonacular et indisponibilite IA ; correction du flag `aiApplied` pour ne plus marquer l'IA comme appliquee quand le rerank echoue, avec `go test ./...`, `npm run lint` et `npx tsc --noEmit` toujours verts
- [x] 2026-04-21 : deblocage du build frontend en remplacant le dossier de sortie par `build/` et en figeant `next build --webpack`, avec `npm run build`, `npm run lint` et `npx tsc --noEmit` valides
- [x] 2026-04-21 : ajout d'un test de fail-safe allergie garantissant le rejet trace d'une recette dangereuse et la conservation d'une alternative sure, plus creation de `doc/security_controls.md` pour formaliser la matrice `menace -> controle -> preuve`
- [x] 2026-04-22 : ajout de tests end-to-end explicites pour profils avec hypertension et medicament de type `statin`, afin de prouver l'effet des regles medicales sur `profile/nutrition`, le filtrage des recommandations et la desactivation du rerank IA ; documentation de la pipeline metier finale dans `doc/business_logic.md`
