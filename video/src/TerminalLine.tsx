import React from "react";
import { interpolate, useCurrentFrame } from "remotion";

const colors: Record<string, string> = {
  green: "#9ece6a",
  red: "#f7768e",
  yellow: "#e0af68",
  gray: "#565f89",
  white: "#c0caf5",
  cyan: "#7dcfff",
  blue: "#7aa2f7",
  magenta: "#bb9af7",
  orange: "#ff9e64",
  dim: "#414868",
};

export type ColorName = keyof typeof colors;

export type TextSegment = {
  text: string;
  color?: ColorName;
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
    ? interpolate(frame - showAtFrame, [0, 4], [0, 1], {
        extrapolateRight: "clamp",
      })
    : 1;

  return (
    <div style={{ opacity, whiteSpace: "pre" }}>
      {segments.map((segment, i) => (
        <span
          key={i}
          style={{
            color: segment.color ? colors[segment.color] : colors.white,
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
}

export const TypingLine: React.FC<TypingLineProps> = ({
  text,
  startFrame,
  typingSpeed = 0.6,
  prefix = [],
}) => {
  const frame = useCurrentFrame();
  if (frame < startFrame) return null;

  const elapsed = frame - startFrame;
  const charsToShow = Math.min(
    Math.floor(elapsed * typingSpeed),
    text.length,
  );
  const isTyping = charsToShow < text.length;
  const showCursor = isTyping || (elapsed - text.length / typingSpeed) % 30 < 15;

  return (
    <div style={{ whiteSpace: "pre" }}>
      {prefix.map((segment, i) => (
        <span
          key={i}
          style={{
            color: segment.color ? colors[segment.color] : colors.white,
            fontWeight: segment.bold ? "bold" : "normal",
          }}
        >
          {segment.text}
        </span>
      ))}
      <span style={{ color: colors.white }}>{text.slice(0, charsToShow)}</span>
      {showCursor && (
        <span
          style={{ backgroundColor: colors.green, color: "#1a1b26" }}
        >
          {" "}
        </span>
      )}
    </div>
  );
};

export const Blank: React.FC<{ showAtFrame: number }> = ({ showAtFrame }) => {
  const frame = useCurrentFrame();
  if (frame < showAtFrame) return null;
  return <div style={{ height: "1.7em" }} />;
};

export { colors };
