import React from "react";
import { AbsoluteFill, Sequence } from "remotion";
import { Terminal } from "./Terminal";
import { TerminalLine, TypingLine, TextSegment } from "./TerminalLine";

const SCENE1_START = 0;
const SCENE2_START = 300;
const SCENE3_START = 750;

const prompt: TextSegment[] = [
  { text: "$ ", color: "green", bold: true },
];

const mysqlPrompt: TextSegment[] = [
  { text: "mysql> ", color: "cyan", bold: true },
];

// Scene 1: mysh add - driver selection
const Scene1: React.FC = () => {
  return (
    <Terminal title="mysh add — driver selection">
      <TypingLine
        text="mysh add"
        startFrame={10}
        typingSpeed={0.6}
        prefix={prompt}
      />

      {/* SSH prompt */}
      <TerminalLine
        segments={[{ text: "Use SSH tunnel? [y/N]: ", color: "white" }]}
        showAtFrame={40}
      />
      <TerminalLine
        segments={[{ text: "N", color: "dim" }]}
        showAtFrame={55}
      />

      {/* DB settings (abbreviated) */}
      <TerminalLine
        segments={[
          { text: "MySQL host [localhost]: ", color: "white" },
          { text: "10.0.0.5", color: "cyan" },
        ]}
        showAtFrame={65}
      />
      <TerminalLine
        segments={[
          { text: "MySQL port [3306]: ", color: "white" },
        ]}
        showAtFrame={75}
      />
      <TerminalLine
        segments={[
          { text: "MySQL user: ", color: "white" },
          { text: "app", color: "cyan" },
        ]}
        showAtFrame={85}
      />
      <TerminalLine
        segments={[{ text: "MySQL password: ********", color: "white" }]}
        showAtFrame={95}
      />
      <TerminalLine
        segments={[
          { text: "Database name: ", color: "white" },
          { text: "legacy_production", color: "cyan" },
        ]}
        showAtFrame={105}
      />

      {/* Environment */}
      <TerminalLine
        segments={[{ text: "Environment:", color: "white" }]}
        showAtFrame={120}
      />
      <TerminalLine
        segments={[{ text: "  1) production", color: "dim" }]}
        showAtFrame={122}
      />
      <TerminalLine
        segments={[{ text: "  2) staging", color: "dim" }]}
        showAtFrame={124}
      />
      <TerminalLine
        segments={[{ text: "  3) development", color: "dim" }]}
        showAtFrame={126}
      />
      <TerminalLine
        segments={[
          { text: "Choice [3]: ", color: "white" },
          { text: "1", color: "cyan" },
        ]}
        showAtFrame={135}
      />

      {/* Driver selection - the highlight */}
      <TerminalLine
        segments={[{ text: "Connection driver:", color: "white", bold: true }]}
        showAtFrame={155}
      />
      <TerminalLine
        segments={[
          { text: "  1) cli    ", color: "dim" },
          { text: "- mysql/mycli command-line client", color: "dim" },
        ]}
        showAtFrame={158}
      />
      <TerminalLine
        segments={[
          { text: "  2) native ", color: "dim" },
          {
            text: "- Go driver (MySQL 4.x old_password 対応)",
            color: "dim",
          },
        ]}
        showAtFrame={161}
      />
      <TypingLine
        text="2"
        startFrame={180}
        typingSpeed={0.3}
        prefix={[{ text: "Choice [1]: ", color: "white" }]}
      />

      {/* Warning */}
      <TerminalLine
        segments={[
          {
            text: "  ⚠ native ドライバは MySQL 4.x の old_password 認証に対応していますが、",
            color: "yellow",
          },
        ]}
        showAtFrame={200}
      />
      <TerminalLine
        segments={[
          {
            text: "    old_password はセキュリティ的に脆弱です。レガシーシステムへの接続用途に限定してください。",
            color: "yellow",
          },
        ]}
        showAtFrame={203}
      />

      {/* Connection name & result */}
      <TerminalLine
        segments={[
          { text: "Connection name: ", color: "white" },
          { text: "legacy-db", color: "cyan" },
        ]}
        showAtFrame={230}
      />
      <TerminalLine
        segments={[
          {
            text: 'Connection "legacy-db" added.',
            color: "green",
            bold: true,
          },
        ]}
        showAtFrame={250}
      />
      <TerminalLine
        segments={[
          { text: 'Connection "legacy-db": OK (45ms)', color: "green" },
        ]}
        showAtFrame={265}
      />
    </Terminal>
  );
};

// Scene 2: mysh connect - REPL
const Scene2: React.FC = () => {
  const tableRows = [
    "+----+-------------+-------+",
    "| id | name        | price |",
    "+----+-------------+-------+",
    "| 1  | Widget Pro  | 2980  |",
    "| 2  | Gadget Mini | 1480  |",
    "| 3  | Sensor Max  | 4500  |",
    "+----+-------------+-------+",
  ];

  return (
    <Terminal title="mysh connect — native REPL">
      <TypingLine
        text="mysh connect legacy-db"
        startFrame={10}
        typingSpeed={0.5}
        prefix={prompt}
      />

      <TerminalLine
        segments={[
          {
            text: "Connected to 10.0.0.5:3306 as app (database: legacy_production)",
            color: "green",
          },
        ]}
        showAtFrame={60}
      />
      <TerminalLine
        segments={[
          {
            text: "Type SQL statements, or 'quit' to exit.",
            color: "dim",
          },
        ]}
        showAtFrame={65}
      />

      {/* SQL query */}
      <TerminalLine segments={[{ text: "", color: "white" }]} showAtFrame={80} />
      <TypingLine
        text="SELECT * FROM products LIMIT 3;"
        startFrame={85}
        typingSpeed={0.45}
        prefix={mysqlPrompt}
      />

      {/* Table output */}
      {tableRows.map((row, i) => (
        <TerminalLine
          key={i}
          segments={[{ text: row, color: i === 1 ? "cyan" : "white" }]}
          showAtFrame={165 + i * 4}
        />
      ))}

      <TerminalLine
        segments={[{ text: "3 rows in set", color: "dim" }]}
        showAtFrame={200}
      />

      {/* quit */}
      <TerminalLine segments={[{ text: "", color: "white" }]} showAtFrame={240} />
      <TypingLine
        text="quit"
        startFrame={250}
        typingSpeed={0.5}
        prefix={mysqlPrompt}
      />
      <TerminalLine
        segments={[{ text: "Bye", color: "dim" }]}
        showAtFrame={270}
      />
    </Terminal>
  );
};

// Scene 3: mysh ping
const Scene3: React.FC = () => {
  return (
    <Terminal title="mysh ping — native driver">
      <TypingLine
        text="mysh ping legacy-db"
        startFrame={10}
        typingSpeed={0.5}
        prefix={prompt}
      />

      <TerminalLine
        segments={[
          {
            text: 'Connection "legacy-db": OK (45ms)',
            color: "green",
            bold: true,
          },
        ]}
        showAtFrame={55}
      />

      <TerminalLine segments={[{ text: "", color: "white" }]} showAtFrame={75} />
      <TypingLine
        text="mysh ping legacy-db"
        startFrame={80}
        typingSpeed={0.5}
        prefix={prompt}
      />

      <TerminalLine
        segments={[
          {
            text: 'Connection "legacy-db": OK (42ms)',
            color: "green",
            bold: true,
          },
        ]}
        showAtFrame={125}
      />
    </Terminal>
  );
};

export const NativeDriverDemo: React.FC = () => {
  return (
    <AbsoluteFill style={{ backgroundColor: "#0d0d1a" }}>
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: 20,
        }}
      >
        <div style={{ width: 760, height: 460 }}>
          <Sequence from={SCENE1_START} durationInFrames={SCENE2_START - SCENE1_START}>
            <Scene1 />
          </Sequence>
          <Sequence from={SCENE2_START} durationInFrames={SCENE3_START - SCENE2_START}>
            <Scene2 />
          </Sequence>
          <Sequence from={SCENE3_START} durationInFrames={900 - SCENE3_START}>
            <Scene3 />
          </Sequence>
        </div>
      </div>
    </AbsoluteFill>
  );
};
