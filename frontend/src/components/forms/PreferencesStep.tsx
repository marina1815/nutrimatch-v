import { IngredientAutocompleteInput } from "@/components/forms/IngredientAutocompleteInput";
import { UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    likes?: string;
    dislikes?: string;
  };
};

export function PreferencesStep({ data, setData, errors }: Props) {
  const likes = data.preferences.likes ?? [];
  const dislikes = data.preferences.dislikes ?? [];

  return (
    <div className="nm-stack">
      <IngredientAutocompleteInput
        label="Aliments aimés"
        placeholder="Cherche: poulet, riz, oeufs..."
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
        helperText="Contrainte souple: ces aliments améliorent le score, sans autoriser une recette dangereuse."
      />

      <IngredientAutocompleteInput
        label="Aliments moins appréciés"
        placeholder="Cherche: brocoli, champignons..."
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
        helperText="Contrainte souple: ces aliments sont pénalisés, jamais traités comme une interdiction."
      />
    </div>
  );
}
