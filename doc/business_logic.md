# NutriMatch Business Logic

Document de reference sur la logique metier finale actuellement implementee.

## 1. Flux fonctionnel cible

Le flux suit l'intention du DFD:

1. l'utilisateur soumet un profil brut
2. le backend valide, normalise et persiste le profil
3. le backend calcule un profil nutritionnel derive
4. l'orchestrateur construit plusieurs plans de recherche
5. Spoonacular fournit des recettes candidates enrichies
6. un pare-feu nutritionnel deterministe filtre les recettes
7. un scoring deterministe classe les recettes restantes
8. l'IA peut seulement reranker a la marge si le profil n'est pas sensible
9. la trace de decision est stockee et restituable

## 2. Construction du profil nutritionnel

Le backend ne se contente pas de stocker les donnees brutes. Il derive un profil nutritionnel a partir de:

- age, sexe, poids, taille
- niveau d'activite
- objectif
- nombre de repas par jour
- preferences et styles de repas
- allergies, exclusions, maladies chroniques, conditions et medicaments
- regles medicales actives

Le calcul derive notamment:

- `BMI`, categorie BMI et `BMR`
- calories estimees puis calories cibles
- macros cibles protein/carbs/fat
- plafonds par repas:
  - calories
  - glucides
  - lipides
  - sucre
  - sodium
- minimum proteique par repas
- exclusions derivees
- styles de repas recommandes
- `matchedRuleCodes` pour expliquer quelles regles medicales ont influence le profil

## 3. Regles medicales

Les regles medicales sont des contraintes deterministes stockees dans `catalog.medical_rules`.

Une regle peut etre activee:

- par `condition_key`
- par `medication_pattern`

Une fois activee, elle peut:

- exclure des ingredients
- exclure des tags
- exiger des tags
- imposer des bornes nutritionnelles plus strictes
- ajouter une rationale metier exploitable

Exemples de logique couverte:

- une hypertension peut abaisser le plafond sodium et imposer des repas `low-sodium`
- un medicament de type `statin` peut exclure `grapefruit`

## 4. Orchestration hybride

Le moteur ne fait pas une seule recherche simple. Il prepare plusieurs plans:

- `strict_profile`
- `goal_balanced`
- `similarity_expansion` si des signaux de similarite existent

Chaque plan combine:

- styles de repas
- ingredients preferes
- exclusions
- intolerances
- cuisines preferees / exclues
- type de repas
- temps maximal
- bornes nutritionnelles derivees

Les reponses Spoonacular sont enrichies, fusionnees par `recipeId`, puis conservees avec leur provenance:

- plans de recherche
- source cache ou upstream
- details d'enrichissement nutrition/ingredients

## 5. Pare-feu nutritionnel deterministe

Le filtrage dur est la barriere de securite principale.

Une recette est rejetee si elle:

- contient un ingredient bloque
- viole une regle medicale active
- depasse un plafond nutritionnel derive
- manque un tag medical obligatoire
- tombe sous un seuil proteique minimal

Ce filtre est prioritaire sur toute autre logique, y compris l'IA.

## 6. Scoring deterministe

Une recette qui passe les filtres durs recoit ensuite un score deterministe.

Le score prend en compte:

- alignement avec les ingredients aimes
- proximite avec des signaux de similarite
- compatibilite avec les styles recommandes
- alignement nutritionnel sur le profil derive

Le score produit aussi:

- des raisons d'acceptation
- une decomposition du score
- une explication metier lisible

## 7. Role de l'IA

L'IA est strictement non autoritative.

Elle:

- ne voit qu'un payload minimise
- ne recoit pas les donnees sante brutes
- ne peut agir que sur un faible bonus/malus de classement
- ne peut jamais ajouter une recette absente
- ne peut jamais contourner les filtres deterministes

L'IA est desactivee pour les profils sensibles, notamment si:

- des regles medicales sont actives
- l'utilisateur declare une maladie chronique
- l'utilisateur prend des medicaments

## 8. Trace et explicabilite

Chaque execution de recommandation conserve:

- un `runId`
- un resume de sources
- un resume de decision
- une trace externe par plan de recherche
- les candidats acceptes et rejetes
- les raisons de rejet et d'acceptation
- les details de score
- la provenance des recettes

Cela aligne l'application sur l'objectif du projet: recommander, mais aussi expliquer et auditer.

## 9. Regles de priorite

L'ordre metier applique aujourd'hui est:

1. contraintes de securite et de sante
2. profil nutritionnel derive
3. filtres de recherche Spoonacular
4. scoring deterministe
5. reranking IA optionnel

Autrement dit:

- une allergie ou une exclusion forte gagne toujours
- une contrainte medicale gagne toujours
- une preference positive ne peut pas annuler une contrainte de securite
- l'IA ne peut pas reintroduire une recette rejetee

## 10. Limitations actuelles

- la similarite utilisateur est encore prudente et partiellement conceptualisee
- la couche vectorielle est preparee mais reste limitee dans son exploitation metier
- les regles medicales doivent continuer a etre enrichies pour couvrir plus de cas cliniques
- l'immutabilite forte de l'audit depend encore de l'infrastructure de production
