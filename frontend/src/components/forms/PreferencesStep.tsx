import { Checkbox } from "@/components/ui/Checkbox";
import { CUISINE_OPTIONS, DIET_OPTIONS, MEAL_STYLE_OPTIONS } from "@/lib/constants";
import { UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    mealsPerDay?: string;
  };
};

export function PreferencesStep({ data, setData, errors }: Props) {
  const mealStyles = data.preferences.mealStyles ?? [];
  const likes = data.preferences.likes ?? [];
  const dislikes = data.preferences.dislikes ?? [];
  const diets = data.preferences.diets ?? [];
  const cuisines = data.preferences.cuisines ?? [];

  const toggleArrayPreference = (
    key: "mealStyles" | "diets" | "cuisines",
    value: string
  ) => {
    setData((prev) => {
      const current = prev.preferences[key] ?? [];
      const exists = current.includes(value);

      return {
        ...prev,
        preferences: {
          ...prev.preferences,
          [key]: exists ? current.filter((v) => v !== value) : [...current, value],
        },
      };
    });
  };

  return (
    <div className="nm-stack">
      <div className="nm-field">
        <label className="nm-label">Aliments aimés</label>
        <input
          className="nm-input"
          placeholder="Poulet, riz, œufs..."
          value={likes.join(", ")}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              preferences: {
                ...prev.preferences,
                likes: e.target.value.split(",").map((item) => item.trim()).filter(Boolean),
              },
            }))
          }
        />
      </div>

      <div className="nm-field">
        <label className="nm-label">Aliments non aimés</label>
        <input
          className="nm-input"
          placeholder="Brocoli, champignons..."
          value={dislikes.join(", ")}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              preferences: {
                ...prev.preferences,
                dislikes: e.target.value.split(",").map((item) => item.trim()).filter(Boolean),
              },
            }))
          }
        />
      </div>

      <div className="nm-field">
        <label className="nm-label">Régimes alimentaires (API)</label>
        <div className="nm-check-grid">
          {DIET_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={diets.includes(item.value)}
              onChange={() => toggleArrayPreference("diets", item.value)}
            />
          ))}
        </div>
      </div>

      <div className="nm-field">
        <label className="nm-label">Cuisines préférées (Spoonacular)</label>
        <div className="nm-check-grid">
          {CUISINE_OPTIONS.map((item) => (
            <Checkbox
              key={item.value}
              label={item.label}
              checked={cuisines.includes(item.value)}
              onChange={() => toggleArrayPreference("cuisines", item.value)}
            />
          ))}
        </div>
      </div>

      <div className="nm-field">
        <label className="nm-label">Style de repas préféré (Vibes)</label>
        <div className="nm-check-grid">
          {MEAL_STYLE_OPTIONS.map((item) => (
            <Checkbox
              key={item}
              label={item}
              checked={mealStyles.includes(item)}
              onChange={() => toggleArrayPreference("mealStyles", item)}
            />
          ))}
        </div>
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