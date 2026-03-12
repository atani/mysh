import React from "react";
import { AbsoluteFill, Sequence } from "remotion";
import { Terminal } from "./Terminal";
import { TerminalLine, TypingLine, Blank, type TextSegment } from "./TerminalLine";
import { SceneTitle } from "./SceneTitle";

const prompt: TextSegment[] = [
  { text: "❯ ", color: "green", bold: true },
];

// ── Scene 1: Basic Flow (frames 0-389) ──
const Scene1: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="1. Quick Start" subtitle="Add → List → Run" showAtFrame={0} duration={50} />
      <Terminal title="mysh — Basic Flow">
        {/* mysh add */}
        <TypingLine text="mysh add" startFrame={55} prefix={prompt} typingSpeed={0.4} />
        <TerminalLine showAtFrame={85} segments={[{ text: "Connection name: ", color: "cyan" }, { text: "production", color: "white" }]} />
        <TerminalLine showAtFrame={95} segments={[{ text: "SSH host: ", color: "cyan" }, { text: "bastion.example.com", color: "white" }]} />
        <TerminalLine showAtFrame={105} segments={[{ text: "DB host: ", color: "cyan" }, { text: "db-01.internal", color: "white" }]} />
        <TerminalLine showAtFrame={115} segments={[{ text: "DB user: ", color: "cyan" }, { text: "app", color: "white" }]} />
        <TerminalLine showAtFrame={125} segments={[{ text: "Master password: ", color: "cyan" }, { text: "••••••••", color: "dim" }]} />
        <TerminalLine showAtFrame={140} segments={[{ text: "✓ ", color: "green", bold: true }, { text: "Connection ", color: "white" }, { text: "production", color: "cyan", bold: true }, { text: " saved", color: "white" }]} />
        <Blank showAtFrame={155} />

        {/* mysh list */}
        <TypingLine text="mysh list" startFrame={165} prefix={prompt} typingSpeed={0.4} />
        <TerminalLine showAtFrame={200} segments={[{ text: "  production  ", color: "cyan", bold: true }, { text: "bastion.example.com → db-01.internal:3306", color: "gray" }]} />
        <Blank showAtFrame={215} />

        {/* mysh run */}
        <TypingLine text='mysh run production -e "SHOW TABLES"' startFrame={225} prefix={prompt} typingSpeed={0.5} />
        <TerminalLine showAtFrame={305} segments={[{ text: "Opening SSH tunnel via deploy@bastion.example.com...", color: "yellow" }]} />
        <TerminalLine showAtFrame={320} segments={[{ text: "Tunnel ready on port 54210", color: "yellow" }]} />
        <TerminalLine showAtFrame={335} segments={[{ text: "+----------------+", color: "white" }]} />
        <TerminalLine showAtFrame={340} segments={[{ text: "| Tables_in_mydb |", color: "white" }]} />
        <TerminalLine showAtFrame={345} segments={[{ text: "+----------------+", color: "white" }]} />
        <TerminalLine showAtFrame={350} segments={[{ text: "| users          |", color: "white" }]} />
        <TerminalLine showAtFrame={355} segments={[{ text: "| orders         |", color: "white" }]} />
        <TerminalLine showAtFrame={360} segments={[{ text: "| products       |", color: "white" }]} />
        <TerminalLine showAtFrame={365} segments={[{ text: "+----------------+", color: "white" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Scene 2: Output Formats (frames 0-499) ──
const Scene2: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="2. Output Formats" subtitle="--format markdown | csv | pdf" showAtFrame={0} duration={50} />
      <Terminal title="mysh — Output Formats">
        {/* markdown */}
        <TypingLine text='mysh run production -e "SELECT * FROM users LIMIT 3" --format markdown' startFrame={55} typingSpeed={0.45} />
        <Blank showAtFrame={220} />
        <TerminalLine showAtFrame={225} segments={[{ text: "| id | name  | email             |", color: "white" }]} />
        <TerminalLine showAtFrame={230} segments={[{ text: "| -- | ----- | ----------------- |", color: "dim" }]} />
        <TerminalLine showAtFrame={235} segments={[{ text: "| 1  | Alice | alice@example.com |", color: "white" }]} />
        <TerminalLine showAtFrame={240} segments={[{ text: "| 2  | Bob   | bob@example.com   |", color: "white" }]} />
        <TerminalLine showAtFrame={245} segments={[{ text: "| 3  | Carol | carol@example.com |", color: "white" }]} />
        <Blank showAtFrame={260} />

        {/* csv + save */}
        <TypingLine text='mysh run production -e "SELECT * FROM users" -f csv -o users.csv' startFrame={275} typingSpeed={0.45} />
        <TerminalLine showAtFrame={425} segments={[{ text: "[mysh] ", color: "blue" }, { text: "saved to users.csv", color: "green" }]} />
        <Blank showAtFrame={440} />

        {/* pdf + save */}
        <TypingLine text='mysh run production -e "SELECT * FROM users" -f pdf -o report.pdf' startFrame={455} typingSpeed={0.45} />
        <TerminalLine showAtFrame={605} segments={[{ text: "[mysh] ", color: "blue" }, { text: "saved to report.pdf", color: "green" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Scene 3: SSH Tunnel + Masking (frames 0-449) ──
const Scene3: React.FC = () => {
  return (
    <AbsoluteFill style={{ padding: 20 }}>
      <SceneTitle title="3. Tunnel & Masking" subtitle="Background tunnels + auto data masking" showAtFrame={0} duration={50} />
      <Terminal title="mysh — SSH Tunnel & Masking">
        {/* tunnel start */}
        <TypingLine text="mysh tunnel production" startFrame={55} prefix={prompt} typingSpeed={0.4} />
        <TerminalLine showAtFrame={120} segments={[{ text: "Opening SSH tunnel via deploy@bastion.example.com...", color: "yellow" }]} />
        <TerminalLine showAtFrame={135} segments={[{ text: "✓ ", color: "green", bold: true }, { text: "Background tunnel started on port ", color: "white" }, { text: "54210", color: "cyan", bold: true }]} />
        <Blank showAtFrame={150} />

        {/* tunnel list */}
        <TypingLine text="mysh tunnel" startFrame={160} prefix={prompt} typingSpeed={0.4} />
        <TerminalLine showAtFrame={195} segments={[{ text: "  production  ", color: "cyan", bold: true }, { text: "localhost:54210 → db-01.internal:3306  ", color: "gray" }, { text: "● active", color: "green" }]} />
        <Blank showAtFrame={215} />

        {/* run with masking */}
        <TypingLine text='mysh run production --mask -e "SELECT * FROM users LIMIT 3"' startFrame={225} prefix={prompt} typingSpeed={0.45} />
        <TerminalLine showAtFrame={365} segments={[{ text: "Reusing background tunnel \"production\" (localhost:54210)", color: "yellow" }]} />
        <TerminalLine showAtFrame={380} segments={[{ text: "[mysh] masking columns: email, phone", color: "blue" }]} />
        <Blank showAtFrame={395} />
        <TerminalLine showAtFrame={400} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={405} segments={[{ text: "| id | name  | email             | phone    |", color: "white" }]} />
        <TerminalLine showAtFrame={410} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
        <TerminalLine showAtFrame={415} segments={[
          { text: "|  1 | Alice | ", color: "white" },
          { text: "a***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={420} segments={[
          { text: "|  2 | Bob   | ", color: "white" },
          { text: "b***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={425} segments={[
          { text: "|  3 | Carol | ", color: "white" },
          { text: "c***@example.com ", color: "red" },
          { text: " | ", color: "white" },
          { text: "0***    ", color: "red" },
          { text: " |", color: "white" },
        ]} />
        <TerminalLine showAtFrame={430} segments={[{ text: "+----+-------+-------------------+----------+", color: "white" }]} />
      </Terminal>
    </AbsoluteFill>
  );
};

// ── Main Composition ──
export const MyshDemo: React.FC = () => {
  return (
    <AbsoluteFill style={{ backgroundColor: "#0f0f14" }}>
      <Sequence from={0} durationInFrames={400}>
        <Scene1 />
      </Sequence>
      <Sequence from={400} durationInFrames={640}>
        <Scene2 />
      </Sequence>
      <Sequence from={1040} durationInFrames={460}>
        <Scene3 />
      </Sequence>
    </AbsoluteFill>
  );
};
