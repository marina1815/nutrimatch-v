import React from "react";

export function Card({ children }: { children: React.ReactNode }) {
  return <div className="nm-card">{children}</div>;
}