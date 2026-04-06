import { Checkbox } from "@/components/ui/Checkbox";
import { MEAL_STYLE_OPTIONS } from "@/lib/constants";
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

  const toggleMealStyle = (item: string) => {
    setData((prev) => {
      const currentMealStyles = prev.preferences.mealStyles ?? [];
      const exists = currentMealStyles.includes(item);

      return {
        ...prev,
        preferences: {
          ...prev.preferences,
          mealStyles: exists
            ? currentMealStyles.filter((value) => value !== item)
            : [...currentMealStyles, item],
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
        <label className="nm-label">Style de repas préféré</label>
        <div className="nm-check-grid">
          {MEAL_STYLE_OPTIONS.map((item) => (
            <Checkbox
              key={item}
              label={item}
              checked={mealStyles.includes(item)}
              onChange={() => toggleMealStyle(item)}
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