import React from "react";

interface TerminalProps {
  children: React.ReactNode;
  title?: string;
}

export const Terminal: React.FC<TerminalProps> = ({
  children,
  title = "Terminal",
}) => {
  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        backgroundColor: "#1a1a2e",
        display: "flex",
        flexDirection: "column",
        fontFamily: "'SF Mono', 'Monaco', 'Menlo', monospace",
        fontSize: 13,
        borderRadius: 10,
        overflow: "hidden",
        boxShadow: "0 20px 60px rgba(0, 0, 0, 0.5)",
      }}
    >
      <div
        style={{
          height: 32,
          backgroundColor: "#2d2d44",
          display: "flex",
          alignItems: "center",
          paddingLeft: 12,
          gap: 8,
        }}
      >
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            backgroundColor: "#ff5f57",
          }}
        />
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            backgroundColor: "#febc2e",
          }}
        />
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            backgroundColor: "#28c840",
          }}
        />
        <span style={{ marginLeft: 8, color: "#888", fontSize: 13 }}>
          {title}
        </span>
      </div>
      <div
        style={{
          flex: 1,
          padding: 16,
          overflowY: "auto",
          lineHeight: 1.7,
        }}
      >
        {children}
      </div>
    </div>
  );
};
