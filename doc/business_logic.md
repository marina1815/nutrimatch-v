# Logique Metier NutriMatch

## Pipeline

1. Charger le profil utilisateur, le profil nutritionnel calcule, les preferences et contraintes.
2. Charger les regles medicales actives.
3. Interroger le catalogue local complet via un pool large.
4. Appliquer les hard constraints: allergies, allergies croisees, ingredients exclus, maladies, medicaments, regles medicales.
5. Calculer un score deterministe avec les soft constraints: aliments aimes/non aimes, cuisines, types de repas, objectif, similarite.
6. Exclure les recettes choisies pendant 7 jours et eviter le set precedent si assez d'alternatives sures existent.
7. Construire un pool sur pour l'IA.
8. Demander a l'IA de choisir et expliquer jusqu'a 20 recettes uniquement dans ce pool.
9. Si l'IA echoue ou sort du contrat, utiliser le tirage deterministe pondere.
10. Persister le set quotidien pendant 24h et reutiliser les explications stockees.

## Hard Constraints

Ces regles sont non negociables:

- allergie declaree;
- allergie croisee;
- ingredient exclu strictement;
- maladie fermee: diabete, hypertension, maladie cardiaque, insuffisance renale, hypercholesterolemie, sensibilite digestive;
- medicament connu par une regle active;
- plafond nutritionnel medical applicable;
- ingredient derive par le profil nutritionnel.

Si moins de 20 recettes sures existent, l'application retourne moins de 20 recettes.

## Soft Constraints

Ces signaux influencent seulement le score:

- aliments aimes;
- aliments non aimes;
- cuisines preferees;
- cuisines a eviter;
- types de repas;
- objectif nutritionnel;
- similarite deterministe/vectorielle.

Une soft constraint ne peut jamais annuler une hard constraint.

## IA

L'IA n'a pas d'autorite medicale. Elle ne peut pas inventer de recette, modifier un score, modifier un ordre de securite, ajouter un ingredient ou valider une substitution dangereuse.

Sortie quotidienne acceptee:

```json
[
  { "mealId": "123", "explanation": "..." }
]
```

Tout champ supplementaire ou ID inconnu invalide la sortie.

## Choix D'Une Recette

`POST /recommendations/:profileId/meals/:mealId/choose` ne fonctionne que sur le set actif. Le choix:

- cree un evenement metier;
- exclut la recette pendant 7 jours;
- peut produire un guide de preparation estime;
- revalide toutes les substitutions IA avant stockage.
