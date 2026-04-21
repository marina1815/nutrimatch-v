import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { SEX_OPTIONS } from "@/lib/constants";
import { Sex, UserProfile } from "@/lib/types";

type Props = {
  data: UserProfile;
  setData: React.Dispatch<React.SetStateAction<UserProfile>>;
  errors?: {
    fullName?: string;
    age?: string;
    sex?: string;
    weight?: string;
    height?: string;
    profession?: string;
    city?: string;
  };
};

export function PersonalInfoStep({ data, setData, errors }: Props) {
  return (
    <>
      <Input
        label="Nom complet"
        value={data.personal.fullName}
        onChange={(e) =>
          setData((prev) => ({
            ...prev,
            personal: { ...prev.personal, fullName: e.target.value },
          }))
        }
        error={errors?.fullName}
      />

      <div className="nm-grid">
        <Input
          label="Age"
          type="number"
          min={10}
          max={120}
          value={data.personal.age}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, age: Number(e.target.value) || "" },
            }))
          }
          error={errors?.age}
        />

        <Select
          label="Sexe"
          value={data.personal.sex}
          options={SEX_OPTIONS}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, sex: e.target.value as Sex },
            }))
          }
          error={errors?.sex}
        />
      </div>

      <div className="nm-grid">
        <Input
          label="Poids (kg)"
          type="number"
          min={20}
          max={400}
          value={data.personal.weight}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, weight: Number(e.target.value) || "" },
            }))
          }
          error={errors?.weight}
        />

        <Input
          label="Taille (cm)"
          type="number"
          min={80}
          max={250}
          value={data.personal.height}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, height: Number(e.target.value) || "" },
            }))
          }
          error={errors?.height}
        />
      </div>

      <div className="nm-grid">
        <Input
          label="Profession"
          maxLength={120}
          value={data.personal.profession}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, profession: e.target.value },
            }))
          }
          error={errors?.profession}
        />

        <Input
          label="Ville"
          maxLength={120}
          value={data.personal.city}
          onChange={(e) =>
            setData((prev) => ({
              ...prev,
              personal: { ...prev.personal, city: e.target.value },
            }))
          }
          error={errors?.city}
        />
      </div>
    </>
  );
}
