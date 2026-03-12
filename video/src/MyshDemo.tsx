import React from "react";
import { AbsoluteFill, Sequence } from "remotion";
import { Terminal } from "./Terminal";
import { TerminalLine, TypingLine, Blank, type TextSegment } from "./TerminalLine";
import { SceneTitle } from "./SceneTitle";

const prompt: TextSegment[] = [
  { text: "❯ ", color: "green", bold: true },
];

// ── Scene 1: Basic Flow (frames 0-249) ──
const Scene1: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="1. Quick Start" subtitle="Add → List → Run" showAtFrame={0} duration={40} />
      <Terminal title="mysh — Basic Flow">
        {/* mysh add */}
        <TypingLine text="mysh add" startFrame={45} prefix={prompt} />
        <TerminalLine showAtFrame={65} segments={[{ text: "Connection name: ", color: "cyan" }, { text: "production", color: "white" }]} />
        <TerminalLine showAtFrame={72} segments={[{ text: "SSH host: ", color: "cyan" }, { text: "bastion.example.com", color: "white" }]} />
        <TerminalLine showAtFrame={79} segments={[{ text: "DB host: ", color: "cyan" }, { text: "db-01.internal", color: "white" }]} />
        <TerminalLine showAtFrame={86} segments={[{ text: "DB user: ", color: "cyan" }, { text: "app", color: "white" }]} />
        <TerminalLine showAtFrame={93} segments={[{ text: "Master password: ", color: "cyan" }, { text: "••••••••", color: "dim" }]} />
        <TerminalLine showAtFrame={100} segments={[{ text: "✓ ", color: "green", bold: true }, { text: "Connection ", color: "white" }, { text: "production", color: "cyan", bold: true }, { text: " saved", color: "white" }]} />
        <Blank showAtFrame={108} />

        {/* mysh list */}
        <TypingLine text="mysh list" startFrame={112} prefix={prompt} />
        <TerminalLine showAtFrame={132} segments={[{ text: "  production  ", color: "cyan", bold: true }, { text: "bastion.example.com → db-01.internal:3306", color: "gray" }]} />
        <Blank showAtFrame={140} />

        {/* mysh run */}
        <TypingLine text='mysh run production -e "SELECT COUNT(*) FROM users"' startFrame={145} typingSpeed={0.8} />
        <TerminalLine showAtFrame={215} segments={[{ text: "Opening SSH tunnel via deploy@bastion.example.com...", color: "yellow" }]} />
        <TerminalLine showAtFrame={225} segments={[{ text: "Tunnel ready on port 54210", color: "yellow" }]} />
        <TerminalLine showAtFrame={233} segments={[{ text: "+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={236} segments={[{ text: "| COUNT(*) |", color: "white" }]} />
        <TerminalLine showAtFrame={239} segments={[{ text: "+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={242} segments={[{ text: "|    ", color: "white" }, { text: "42,891", color: "green", bold: true }, { text: " |", color: "white" }]} />
        <TerminalLine showAtFrame={245} segments={[{ text: "+----------+", color: "white" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Scene 2: Output Formats (frames 0-249) ──
const Scene2: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="2. Output Formats" subtitle="--format markdown | csv | pdf" showAtFrame={0} duration={40} />
      <Terminal title="mysh — Output Formats">
        {/* markdown */}
        <TypingLine text='mysh run production -e "SELECT * FROM users LIMIT 3" --format markdown' startFrame={45} typingSpeed={0.9} />
        <Blank showAtFrame={130} />
        <TerminalLine showAtFrame={133} segments={[{ text: "| id | name  | email             |", color: "white" }]} />
        <TerminalLine showAtFrame={136} segments={[{ text: "| -- | ----- | ----------------- |", color: "dim" }]} />
        <TerminalLine showAtFrame={139} segments={[{ text: "| 1  | Alice | alice@example.com |", color: "white" }]} />
        <TerminalLine showAtFrame={142} segments={[{ text: "| 2  | Bob   | bob@example.com   |", color: "white" }]} />
        <TerminalLine showAtFrame={145} segments={[{ text: "| 3  | Carol | carol@example.com |", color: "white" }]} />
        <Blank showAtFrame={153} />

        {/* csv + save */}
        <TypingLine text='mysh run production -e "SELECT * FROM users" --format csv -o users.csv' startFrame={158} typingSpeed={0.9} />
        <TerminalLine showAtFrame={245} segments={[{ text: "[mysh] ", color: "blue" }, { text: "saved to users.csv", color: "green" }]} />
        <Blank showAtFrame={253} />

        {/* pdf + save */}
        <TypingLine text='mysh run production -e "SELECT * FROM users" --format pdf -o report.pdf' startFrame={258} typingSpeed={0.9} />
        <TerminalLine showAtFrame={345} segments={[{ text: "[mysh] ", color: "blue" }, { text: "saved to report.pdf", color: "green" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Scene 3: SSH Tunnel + Masking (frames 0-249) ──
const Scene3: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="3. Tunnel & Masking" subtitle="Background tunnels + auto data masking" showAtFrame={0} duration={40} />
      <Terminal title="mysh — SSH Tunnel & Masking">
        {/* tunnel start */}
        <TypingLine text="mysh tunnel production" startFrame={45} prefix={prompt} />
        <TerminalLine showAtFrame={80} segments={[{ text: "Opening SSH tunnel via deploy@bastion.example.com...", color: "yellow" }]} />
        <TerminalLine showAtFrame={90} segments={[{ text: "✓ ", color: "green", bold: true }, { text: "Background tunnel started on port ", color: "white" }, { text: "54210", color: "cyan", bold: true }]} />
        <Blank showAtFrame={98} />

        {/* tunnel list */}
        <TypingLine text="mysh tunnel" startFrame={102} prefix={prompt} />
        <TerminalLine showAtFrame={125} segments={[{ text: "  production  ", color: "cyan", bold: true }, { text: "localhost:54210 → db-01.internal:3306  ", color: "gray" }, { text: "● active", color: "green" }]} />
        <Blank showAtFrame={133} />

        {/* run with masking */}
        <TypingLine text='mysh run production --mask -e "SELECT * FROM users LIMIT 3"' startFrame={138} typingSpeed={0.8} />
        <TerminalLine showAtFrame={215} segments={[{ text: "Reusing background tunnel \"production\" (localhost:54210)", color: "yellow" }]} />
        <TerminalLine showAtFrame={222} segments={[{ text: "[mysh] masking columns: email, phone", color: "blue" }]} />
        <Blank showAtFrame={228} />
        <TerminalLine showAtFrame={230} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={233} segments={[{ text: "| id | name  | email             | phone    |", color: "white" }]} />
        <TerminalLine showAtFrame={236} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={239} segments={[
          { text: "|  1 | Alice | ", color: "white" },
          { text: "a***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={242} segments={[
          { text: "|  2 | Bob   | ", color: "white" },
          { text: "b***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={245} segments={[
          { text: "|  3 | Carol | ", color: "white" },
          { text: "c***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={248} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Main Composition ──
export const MyshDemo: React.FC = () => {
  return (
    <AbsoluteFill style={{ backgroundColor: "#0f0f14" }}>
      <Sequence from={0} durationInFrames={270}>
        <Scene1 />
      </Sequence>
      <Sequence from={270} durationInFrames={370}>
        <Scene2 />
      </Sequence>
      <Sequence from={640} durationInFrames={260}>
        <Scene3 />
      </Sequence>
    </AbsoluteFill>
  );
};
