import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { SEX_OPTIONS } from "@/lib/constants";
import { UserProfile } from "@/lib/types";

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
          label="Âge"
          type="number"
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
              personal: { ...prev.personal, sex: e.target.value as any },
            }))
          }
          error={errors?.sex}
        />
      </div>

      <div className="nm-grid">
        <Input
          label="Poids (kg)"
          type="number"
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