import React from "react";
import { interpolate, useCurrentFrame } from "remotion";

interface SceneTitleProps {
  title: string;
  subtitle?: string;
  showAtFrame: number;
  duration?: number;
}

export const SceneTitle: React.FC<SceneTitleProps> = ({
  title,
  subtitle,
  showAtFrame,
  duration = 45,
}) => {
  const frame = useCurrentFrame();
  if (frame < showAtFrame || frame >= showAtFrame + duration) return null;

  const progress = frame - showAtFrame;
  const opacity = interpolate(
    progress,
    [0, 8, duration - 10, duration],
    [0, 1, 1, 0],
    { extrapolateRight: "clamp" },
  );

  const translateY = interpolate(progress, [0, 8], [10, 0], {
    extrapolateRight: "clamp",
  });

  return (
    <div
      style={{
        position: "absolute",
        top: 0,
        left: 0,
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        backgroundColor: "#1a1b26",
        opacity,
        transform: `translateY(${translateY}px)`,
        zIndex: 10,
      }}
    >
      <div
        style={{
          color: "#7aa2f7",
          fontSize: 32,
          fontWeight: "bold",
          fontFamily: "'SF Mono', 'Monaco', 'Menlo', monospace",
        }}
      >
        {title}
      </div>
      {subtitle && (
        <div
          style={{
            color: "#565f89",
            fontSize: 16,
            marginTop: 8,
            fontFamily: "'SF Mono', 'Monaco', 'Menlo', monospace",
          }}
        >
          {subtitle}
        </div>
      )}
    </div>
  );
};
