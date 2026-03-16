import React from "react";
import { interpolate, useCurrentFrame } from "remotion";

const colors: Record<string, string> = {
  green: "#22c55e",
  red: "#ef4444",
  yellow: "#eab308",
  gray: "#6b7280",
  white: "#e5e5e5",
  cyan: "#06b6d4",
  blue: "#3b82f6",
  dim: "#555570",
};

export type TextSegment = {
  text: string;
  color?: string;
  bold?: boolean;
};

interface TerminalLineProps {
  segments: TextSegment[];
  showAtFrame: number;
  fadeIn?: boolean;
}

export const TerminalLine: React.FC<TerminalLineProps> = ({
  segments,
  showAtFrame,
  fadeIn = true,
}) => {
  const frame = useCurrentFrame();
  if (frame < showAtFrame) return null;

  const opacity = fadeIn
    ? interpolate(frame - showAtFrame, [0, 5], [0, 1], {
        extrapolateRight: "clamp",
      })
    : 1;

  return (
    <div style={{ opacity, whiteSpace: "pre" }}>
      {segments.map((segment, i) => (
        <span
          key={i}
          style={{
            color: segment.color ? colors[segment.color] || segment.color : colors.white,
            fontWeight: segment.bold ? "bold" : "normal",
          }}
        >
          {segment.text}
        </span>
      ))}
    </div>
  );
};

interface TypingLineProps {
  text: string;
  startFrame: number;
  typingSpeed?: number;
  prefix?: TextSegment[];
  cursorColor?: string;
}

export const TypingLine: React.FC<TypingLineProps> = ({
  text,
  startFrame,
  typingSpeed = 0.5,
  prefix = [],
  cursorColor = "green",
}) => {
  const frame = useCurrentFrame();
  if (frame < startFrame) return null;

  const elapsed = frame - startFrame;
  const charsToShow = Math.min(
    Math.floor(elapsed * typingSpeed),
    text.length,
  );
  const isTyping = charsToShow < text.length;
  const showCursor =
    isTyping || (elapsed - text.length / typingSpeed) % 30 < 15;

  return (
    <div style={{ whiteSpace: "pre" }}>
      {prefix.map((segment, i) => (
        <span
          key={i}
          style={{
            color: segment.color ? colors[segment.color] || segment.color : colors.white,
            fontWeight: segment.bold ? "bold" : "normal",
          }}
        >
          {segment.text}
        </span>
      ))}
      <span style={{ color: colors.white }}>
        {text.slice(0, charsToShow)}
      </span>
      {showCursor && (
        <span
          style={{
            backgroundColor: colors[cursorColor] || cursorColor,
            color: "#1a1a2e",
          }}
        >
          {" "}
        </span>
      )}
    </div>
  );
};

export { colors };
