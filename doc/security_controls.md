# Controles De Securite

## Frontieres De Confiance

- Navigateur utilisateur -> frontend Next.js.
- Frontend -> API Go.
- API -> PostgreSQL.
- API -> Redis.
- API -> Google AI optionnel.

Le catalogue recette est local. Aucune API recette externe n'est dans le chemin critique.

## Matrice Menace Controle Preuve

| Menace | Controle | Preuve |
|---|---|---|
| IDOR/BOLA | Auth middleware, verification proprietaire profil/session | tests routes profil/recommandations |
| CSRF | token signe lie session + origine de confiance | middleware CSRF, tests routes |
| Vol session | refresh token HttpOnly, rotation, revocation, idle timeout | auth service/session repo |
| Bruteforce | auth failures haches + rate limit partage | security rate buckets |
| Injection payload | DTO stricts, refus champs inconnus, limites taille | handlers/validation |
| Allergie contournee | hard filters avant IA, allergies croisees | recommendation service/tests |
| Maladie/medicament contourne | regles medicales deterministes | medical rules + nutrition profile |
| IA malveillante | schema JSON strict, IDs autorises seulement, refus des sorties invalides | tests safe-pool IA |
| Fuite donnees sante | chiffrement repos, prompt minimise, logs bornes | crypto services/audit |
| Audit falsifie | hash-chain append-only | audit repository |
| Surface Docker locale | ports bornes, services separes, healthchecks | docker-compose.yml |

## Politique IA

- L'IA ne voit que des recettes deja sures.
- Elle ne recoit pas de donnees sante brutes; le prompt est limite au pool de recettes deja validees.
- Elle ne peut produire que `mealId` et `explanation` pour le set quotidien.
- Les substitutions de recette choisie sont revalidees avant stockage.

## Limites Residuelle

- Le Docker local reste en HTTP pour le developpement.
- TLS, gestion cloud des secrets, SIEM externe et WAF ne sont pas inclus dans cette passe.
- Les macros locales sont estimees; les decisions critiques reposent sur ingredients/regles plutot que sur ces estimations seules.
