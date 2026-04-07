# **Présentation de "NutriMatch"**

**L'idée générale**

L'idée de notre projet est de créer une application qui aide les utilisateurs à trouver des idées de repas adaptées à leur profil nutritionnel. Au lieu de proposer des recettes au hasard, l'application prend en compte trois piliers :

- **Les préférences :** Ce que l'utilisateur aime ou n'aime pas manger.
- **Le mode de vie :** Son niveau d'activité physique, son âge, sa morphologie et ses objectifs.
- **Les contraintes alimentaires :** Ses allergies ou ses problèmes de santé.

**Comment ça fonctionne ?**

Le projet suit un parcours simple mais robuste pour garantir des résultats de qualité :

- **Récupération des données :** L'utilisateur remplit un formulaire détaillé rassemblant certaines données personnelles (sexe, age, poids, taille), son mode de vie, ses objectifs, ses préférences et ses contraintes alimentaires. Les données seront d'abord vérifiées et validées avant d'être traitées.
- **Calculs et filtres :** Le programme calcul les besoins de la personne et prépare des filtres de sécurité afin de créer le profil nutritionnel de l'utilisateur en incluant les préférences et les contraintes alimentaires.
- **Traitement hybride :** Des bases de données liées à des API vont utiliser des filtres de recherche pour renvoyer des résultats proches des critères demandés, en y ajoutant des préférences d'utilisateurs similaires. Ensuite, un traitement final par un modèle d'IA va affiner et approfondir les suggestions. Bien sûr, les résultats finaux seront filtrés pour être sûr que les critères initiaux sont respectés (Fail-Safe).
- **Affichage :** L'utilisateur reçoit une liste de recommandations de repas personnalisées selon ses préférences, besoins et ses contraintes alimentaires.

**L'objectif final**

**NutriMatch** est un outil qui fait gagner du temps et qui aide les personnes ayant des régimes particuliers à trouver les repas les plus adaptés. C'est une application qui s'adapte à l'humain et non l'inverse.