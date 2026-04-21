import { Checkbox } from "@/components/ui/Checkbox";
import { IngredientAutocompleteInput } from "@/components/forms/IngredientAutocompleteInput";
import { CUISINE_OPTIONS, MEAL_STYLE_OPTIONS, MEAL_TYPE_OPTIONS } from "@/lib/constants";
import { Cuisine, MealType, UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    likes?: string;
    dislikes?: string;
    mealStyles?: string;
    mealTypes?: string;
    preferredCuisines?: string;
    excludedCuisines?: string;
    mealsPerDay?: string;
  };
};

export function PreferencesStep({ data, setData, errors }: Props) {
  const mealStyles = data.preferences.mealStyles ?? [];
  const likes = data.preferences.likes ?? [];
  const dislikes = data.preferences.dislikes ?? [];
  const mealTypes = data.preferences.mealTypes ?? [];
  const preferredCuisines = data.preferences.preferredCuisines ?? [];
  const excludedCuisines = data.preferences.excludedCuisines ?? [];

  const toggleMealStyle = (value: string) => {
    setData((prev) => {
      const currentMealStyles = prev.preferences.mealStyles ?? [];
      const exists = currentMealStyles.includes(value as typeof currentMealStyles[number]);

      return {
        ...prev,
        preferences: {
          ...prev.preferences,
          mealStyles: exists
            ? currentMealStyles.filter((item) => item !== value)
            : [...currentMealStyles, value as typeof currentMealStyles[number]],
        },
      };
    });
  };

  const toggleMealType = (value: string) => {
    setData((prev) => {
      const current = prev.preferences.mealTypes ?? [];
      const typedValue = value as MealType;
      const exists = current.includes(typedValue);

      return {
        ...prev,
        preferences: {
          ...prev.preferences,
          mealTypes: exists
            ? current.filter((item) => item !== typedValue)
            : [...current, typedValue],
        },
      };
    });
  };

  const toggleCuisine = (section: "preferredCuisines" | "excludedCuisines", value: string) => {
    setData((prev) => {
      const current = prev.preferences[section] ?? [];
      const typedValue = value as Cuisine;
      const exists = current.includes(typedValue);

      return {
        ...prev,
        preferences: {
          ...prev.preferences,
          [section]: exists
            ? current.filter((item) => item !== typedValue)
            : [...current, typedValue],
        },
      };
    });
  };

  return (
    <div className="nm-stack">
      <IngredientAutocompleteInput
        label="Aliments aimes"
        placeholder="Poulet, riz, oeufs..."
        values={likes}
        onChange={(nextValues) =>
          setData((prev) => ({
            ...prev,
            preferences: {
              ...prev.preferences,
              likes: nextValues,
            },
          }))
        }
        error={errors?.likes}
        maxItems={25}
      />

      <IngredientAutocompleteInput
        label="Aliments non aimes"
        placeholder="Brocoli, champignons..."
        values={dislikes}
        onChange={(nextValues) =>
          setData((prev) => ({
            ...prev,
            preferences: {
              ...prev.preferences,
              dislikes: nextValues,
            },
          }))
        }
        error={errors?.dislikes}
        maxItems={25}
      />

      <div className="nm-field">
        <label className="nm-label">Style de repas prefere</label>
        <div className="nm-check-grid">
          {MEAL_STYLE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={mealStyles.includes(item.value)}
              onChange={() => toggleMealStyle(item.value)}
            />
          ))}
        </div>
        {errors?.mealStyles && <span className="nm-error">{errors.mealStyles}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Types de repas preferes</label>
        <div className="nm-check-grid">
          {MEAL_TYPE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={mealTypes.includes(item.value)}
              onChange={() => toggleMealType(item.value)}
            />
          ))}
        </div>
        {errors?.mealTypes && <span className="nm-error">{errors.mealTypes}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Cuisines preferees</label>
        <div className="nm-check-grid">
          {CUISINE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={preferredCuisines.includes(item.value)}
              onChange={() => toggleCuisine("preferredCuisines", item.value)}
            />
          ))}
        </div>
        {errors?.preferredCuisines && <span className="nm-error">{errors.preferredCuisines}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Cuisines a exclure</label>
        <div className="nm-check-grid">
          {CUISINE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={excludedCuisines.includes(item.value)}
              onChange={() => toggleCuisine("excludedCuisines", item.value)}
            />
          ))}
        </div>
        {errors?.excludedCuisines && <span className="nm-error">{errors.excludedCuisines}</span>}
      </div>

      <div className="nm-field">
        <label className="nm-label">Combien de repas par jour ?</label>
        <input
          className={`nm-input ${errors?.mealsPerDay ? "nm-input-error" : ""}`}
          type="number"
          min={1}
          max={8}
          value={data.preferences.mealsPerDay ?? ""}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              preferences: {
                ...prev.preferences,
                mealsPerDay: Number(e.target.value) || "",
              },
            }))
          }
        />
        {errors?.mealsPerDay && <span className="nm-error">{errors.mealsPerDay}</span>}
      </div>
    </div>
  );
}
