# Local Catalog Contract

NutriMatch utilise le catalogue local comme source recette primaire et obligatoire.

## Sources

- `recette_EN.xlsx`: recettes.
- `ingrediant_EN.xlsx` et `ing.xlsx`: ingredients et alias.
- `liste des allergies.xlsx`: allergies et ingredients associes.
- `croise.xlsx`: allergies croisees.

Les fichiers sont importes avec normalisation:

- trim des cellules texte;
- suppression logique des lignes vides;
- deduplication par cle canonique;
- relation recette -> ingredients;
- relation ingredient -> allergie;
- relation ingredient -> ingredient croise.

## Contrat Backend

Les repositories exposent des types internes:

- `LocalRecipeCandidate`;
- `LocalRecipeQuery`;
- `CatalogOption`;
- modeles persistants `catalog.local_recipes`, `catalog.local_ingredients`, `catalog.local_recipe_ingredients`, `catalog.local_allergies`, `catalog.local_ingredient_allergies`, `catalog.local_cross_allergies`.

## Securite Nutritionnelle

Le catalogue local ne decide pas seul qu'une recette est sure. Le moteur applique ensuite:

- allergies declarees;
- allergies croisees;
- ingredients exclus;
- maladies fermees;
- medicaments;
- regles medicales actives;
- plafonds nutritionnels quand ils sont applicables.

Les macros locales sont marquees `estimated`. Les hard constraints ne sont jamais relachees pour obtenir 20 recettes.

## Recommandations Quotidiennes

Le backend evalue le catalogue local complet ou un pool large configurable, puis produit jusqu'a 20 recettes sures pendant 24h.

Une recette choisie est exclue pendant 7 jours. Les recettes du set precedent sont evitees si assez d'alternatives sures existent.

## IA

L'IA recoit uniquement des IDs deja valides et des faits minimises. Elle peut choisir/expliquer parmi le pool sur, mais toute sortie hors contrat est ignoree.
