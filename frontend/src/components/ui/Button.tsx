import React from "react";

type Props = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary";
};

export function Button({ variant = "primary", className = "", ...props }: Props) {
  return (
    <button
      {...props}
      className={`nm-btn ${variant === "secondary" ? "nm-btn-secondary" : "nm-btn-primary"} ${className}`}
    />
  );
}