import React from "react";

type Option = { value: string; label: string };

type Props = React.SelectHTMLAttributes<HTMLSelectElement> & {
  label: string;
  options: Option[];
  error?: string;
};

export function Select({ label, options, error, ...props }: Props) {
  const generatedID = React.useId();
  const id = props.id ?? generatedID;

  return (
    <div className="nm-field">
      <label className="nm-label" htmlFor={id}>{label}</label>
      <select id={id} className={`nm-input ${error ? "nm-input-error" : ""}`} {...props}>
        <option value="">Select...</option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && <span className="nm-error">{error}</span>}
    </div>
  );
}
