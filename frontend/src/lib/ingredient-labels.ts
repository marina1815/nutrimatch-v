const INGREDIENT_LABELS_FR: Record<string, string> = {
  almond: "Amande",
  apple: "Pomme",
  avocado: "Avocat",
  banana: "Banane",
  basil: "Basilic",
  beef: "Boeuf",
  "bell pepper": "Poivron",
  broccoli: "Brocoli",
  "brown rice": "Riz complet",
  butter: "Beurre",
  carrot: "Carotte",
  cheese: "Fromage",
  chicken: "Poulet",
  "chicken breast": "Blanc de poulet",
  chickpea: "Pois chiche",
  cinnamon: "Cannelle",
  cod: "Cabillaud",
  cream: "Creme",
  cucumber: "Concombre",
  egg: "Oeuf",
  eggs: "Oeufs",
  flour: "Farine",
  garlic: "Ail",
  ginger: "Gingembre",
  "greek yogurt": "Yaourt grec",
  "green bean": "Haricot vert",
  honey: "Miel",
  lamb: "Agneau",
  lentil: "Lentille",
  lettuce: "Laitue",
  milk: "Lait",
  mushroom: "Champignon",
  oat: "Avoine",
  "olive oil": "Huile d'olive",
  onion: "Oignon",
  pasta: "Pates",
  peanut: "Arachide",
  peas: "Petits pois",
  pepper: "Poivre",
  pork: "Porc",
  potato: "Pomme de terre",
  quinoa: "Quinoa",
  rice: "Riz",
  salmon: "Saumon",
  salt: "Sel",
  shrimp: "Crevette",
  spinach: "Epinard",
  sugar: "Sucre",
  "sweet potato": "Patate douce",
  tofu: "Tofu",
  tomato: "Tomate",
  tuna: "Thon",
  turkey: "Dinde",
  walnut: "Noix",
  wheat: "Ble",
  "whole wheat": "Ble complet",
  yogurt: "Yaourt",
};

const TERM_LABELS_FR: Record<string, string> = {
  anchovy: "anchois",
  asparagus: "asperge",
  beans: "haricots",
  bean: "haricot",
  bread: "pain",
  breast: "blanc",
  cabbage: "chou",
  cake: "gateau",
  cauliflower: "chou-fleur",
  chili: "piment",
  chocolate: "chocolat",
  coconut: "noix de coco",
  corn: "mais",
  crab: "crabe",
  dried: "seche",
  duck: "canard",
  fish: "poisson",
  fried: "frit",
  fruit: "fruit",
  green: "vert",
  grilled: "grille",
  liver: "foie",
  lobster: "homard",
  meat: "viande",
  minced: "hache",
  oil: "huile",
  orange: "orange",
  powder: "poudre",
  red: "rouge",
  sauce: "sauce",
  seed: "graine",
  seeds: "graines",
  sesame: "sesame",
  soup: "soupe",
  spice: "epice",
  spices: "epices",
  sweet: "doux",
  veal: "veau",
  vegetable: "legume",
  vegetables: "legumes",
  white: "blanc",
  whole: "complet",
};

export function getIngredientLabel(value: string): string {
  const normalized = normalizeIngredientValue(value);
  if (!normalized) {
    return "";
  }

  const exact = INGREDIENT_LABELS_FR[normalized];
  if (exact) {
    return exact;
  }

  const translated = translateKnownTerms(normalized);
  return titleCase(translated || normalized);
}

function normalizeIngredientValue(value: string): string {
  return value.replace(/[_-]+/g, " ").replace(/\s+/g, " ").trim().toLowerCase();
}

function translateKnownTerms(value: string): string {
  let translated = value;
  const terms = Object.keys(TERM_LABELS_FR).sort((left, right) => right.length - left.length);
  for (const term of terms) {
    translated = translated.replace(new RegExp(`\\b${escapeRegExp(term)}\\b`, "g"), TERM_LABELS_FR[term]);
  }
  return translated;
}

function titleCase(value: string): string {
  return value
    .split(" ")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
